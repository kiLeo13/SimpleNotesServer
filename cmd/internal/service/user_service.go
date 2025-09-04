package service

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
	"net/http"
	"simplenotes/internal/domain/entity"
	cognitoclient "simplenotes/internal/infrastructure/aws/cognito"
)

type UserRepository interface {
	FindAllInIDs(ids []int) ([]*entity.User, error)
	FindByID(id int) (*entity.User, error)
	Save(user *entity.User) error
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=2,max=80"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64,x-password"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type QueryUsersRequest struct {
	IDs []int `json:"ids" validate:"required,min=1,max=100"`
}

type ConfirmSignupRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,min=1,max=2048"` // 2048????? Just respecting AWS' docs 🤷‍♂️
}

type ResendConfirmRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type UserLoginResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type UserService struct {
	UserRepo UserRepository
	Validate *validator.Validate
	Cognito  cognitoclient.CognitoInterface
}

func NewUserService(userRepo UserRepository, validate *validator.Validate, cogClient cognitoclient.CognitoInterface) *UserService {
	return &UserService{UserRepo: userRepo, Validate: validate, Cognito: cogClient}
}

func (u *UserService) QueryUsers(req *QueryUsersRequest) ([]*UserResponse, *APIError) {
	if err := u.Validate.Struct(req); err != nil {
		return nil, NewError(http.StatusBadRequest, err.Error())
	}

	users, err := u.UserRepo.FindAllInIDs(req.IDs)
	if err != nil {
		log.Errorf("failed to fetch users: %v", err)
		return nil, InternalServerError
	}

	resp := make([]*UserResponse, len(users))
	for i, user := range users {
		resp[i] = toUserResponse(user)
	}
	return resp, nil
}

func (u *UserService) GetUser(id int) (*UserResponse, *APIError) {
	if id < 1 {
		return nil, InvalidIDError
	}

	user, err := u.UserRepo.FindByID(id)
	if err != nil {
		log.Errorf("failed to fetch user for ID %d: %v", id, err)
		return nil, InternalServerError
	}

	if user == nil {
		return nil, NotFoundError
	}

	resp := toUserResponse(user)
	return resp, nil
}

// CreateUser creates a new user on Cognito (as well as in our database),
// and sends a verification code to the user's email address.
func (u *UserService) CreateUser(req *CreateUserRequest) *APIError {
	if err := u.Validate.Struct(req); err != nil {
		return NewError(http.StatusBadRequest, err.Error())
	}

	cogUser := &cognitoclient.User{Email: req.Email, Password: req.Password}
	uuid, apierr, revert := handleUserSignup(u.Cognito, cogUser)
	if apierr != nil {
		return apierr
	}

	// This is our user, in our database <3
	now := NowUTC()
	user := &entity.User{
		SubUUID:       uuid,
		Username:      req.Username,
		EmailVerified: false,
		IsAdmin:       false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := u.UserRepo.Save(user)
	if err != nil {
		// Well... for our case, I have no idea how can SQLite fail here,
		// but better safe than sorry?
		revert()
		log.Errorf("failed to create user: %v", err)
		return InternalServerError
	}
	return nil
}

func (u *UserService) Login(req *UserLoginRequest) (*UserLoginResponse, *APIError) {
	if err := u.Validate.Struct(req); err != nil {
		return nil, NewError(http.StatusBadRequest, err.Error())
	}
	credentials := &cognitoclient.UserLogin{
		Email:    req.Email,
		Password: req.Password,
	}

	auth, apierr := handleUserSignin(u.Cognito, credentials)
	if apierr != nil {
		return nil, apierr
	}
	return &UserLoginResponse{AccessToken: auth.AccessToken, IDToken: auth.IDToken}, nil
}

func (u *UserService) ConfirmSignup(req *ConfirmSignupRequest) *APIError {
	if err := u.Validate.Struct(req); err != nil {
		return NewError(http.StatusBadRequest, err.Error())
	}
	confirms := &cognitoclient.UserConfirmation{
		Email: req.Email,
		Code:  req.Code,
	}

	apierr := handleSignupConfirmation(u.Cognito, confirms)
	if apierr != nil {
		return apierr
	}
	return nil
}

func (u *UserService) ResendConfirmation(req *ResendConfirmRequest) *APIError {
	if err := u.Validate.Struct(req); err != nil {
		return NewError(http.StatusBadRequest, err.Error())
	}

	apierr := handleConfirmResend(u.Cognito, req.Email)
	if apierr != nil {
		return apierr
	}
	return nil
}

// handleUserSignup attempts to register a new user account in Amazon Cognito.
//
// On success, it returns the user's unique identifier (SUB) as a string, and a nil *APIError.
//
// On failure, it returns a non-nil *APIError that describes the reason for the failure—
// such as invalid input or a Cognito-specific error. This error can be safely returned
// to the client to provide feedback.
//
// In all cases, this method also returns a `revert()` function you can call to request
// Cognito for the user deletion from the pool.
//
// Parameters:
//   - req: a pointer to a cognitoclient.User struct containing the user's signup information.
func handleUserSignup(cogClient cognitoclient.CognitoInterface, req *cognitoclient.User) (string, *APIError, func()) {
	revert := func() {
		_ = cogClient.AdminDeleteUser(req.Email)
	}

	uuid, err := cogClient.SignUp(req)
	if err == nil {
		return uuid, nil, revert
	}

	switch {
	case errors.Is(err, &types.InvalidPasswordException{}):
		return "", IDPInvalidPasswordError, revert

	case errors.Is(err, &types.UsernameExistsException{}):
		return "", IDPExistingEmailError, revert

	default:
		log.Errorf("failed to signup user: %v", err)
		return "", InternalServerError, revert
	}
}

func handleUserSignin(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserLogin) (*cognitoclient.AuthCreate, *APIError) {
	auth, err := cogClient.SignIn(req)
	if err == nil {
		return auth, nil
	}

	switch {
	case errors.Is(err, &types.UserNotFoundException{}):
		return nil, IDPUserNotFoundError

	case errors.Is(err, &types.UserNotConfirmedException{}):
		return nil, IDPUserNotConfirmedError

	case errors.Is(err, &types.NotAuthorizedException{}):
		return nil, IDPCredentialsMismatchError

	default:
		log.Errorf("failed to signin user (%s): %v", req.Email, err)
		return nil, InternalServerError
	}
}

func handleSignupConfirmation(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserConfirmation) *APIError {
	err := cogClient.ConfirmAccount(req)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, &types.CodeMismatchException{}):
		return IDPConfirmCodeMismatchError

	case errors.Is(err, &types.ExpiredCodeException{}):
		return IDPConfirmCodeExpiredError

	case errors.Is(err, &types.UserNotFoundException{}):
		return IDPUserNotFoundError

	default:
		log.Errorf("failed to confirm user (%s): %v", req.Email, err)
		return InternalServerError
	}
}

func handleConfirmResend(cogClient cognitoclient.CognitoInterface, email string) *APIError {
	err := cogClient.ResendConfirmation(email)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, &types.UserNotFoundException{}):
		return IDPUserNotFoundError

	case errors.Is(err, &types.InvalidParameterException{}):
		return IDPInvalidParameterError

	default:
		log.Errorf("failed to resend confirmation code to email (%s): %v", email, err)
		return InternalServerError
	}
}

func toUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
