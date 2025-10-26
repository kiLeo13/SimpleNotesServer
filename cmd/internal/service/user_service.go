package service

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
	"simplenotes/cmd/internal/domain/entity"
	cognitoclient "simplenotes/cmd/internal/infrastructure/aws/cognito"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strconv"
)

type UserRepository interface {
	FindAllInIDs(ids []int) ([]*entity.User, error)
	FindByID(id int) (*entity.User, error)
	FindBySub(sub string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	ExistsByEmail(email string) (bool, error)
	Save(user *entity.User) error
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=2,max=80"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64,hasspecial,hasdigit,hasupper,haslower"`
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

type UserExistsRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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

func (u *UserService) QueryUsers(req *QueryUsersRequest) ([]*UserResponse, apierror.ErrorResponse) {
	if err := u.Validate.Struct(req); err != nil {
		return nil, apierror.FromValidationError(err)
	}

	users, err := u.UserRepo.FindAllInIDs(req.IDs)
	if err != nil {
		log.Errorf("failed to fetch users: %v", err)
		return nil, apierror.InternalServerError
	}

	resp := make([]*UserResponse, len(users))
	for i, user := range users {
		resp[i] = toUserResponse(user)
	}
	return resp, nil
}

func (u *UserService) GetUser(token, rawId string) (*UserResponse, apierror.ErrorResponse) {
	tokenData, err := utils.ParseTokenData(token)
	if err != nil {
		return nil, apierror.InvalidAuthTokenError
	}

	user, apierr := u.fetchUser(rawId, tokenData.Sub)
	if apierr != nil {
		return nil, apierr
	}

	if user == nil {
		return nil, apierror.NotFoundError
	}

	resp := toUserResponse(user)
	return resp, nil
}

func (u *UserService) CheckEmail(req *UserExistsRequest) (bool, apierror.ErrorResponse) {
	utils.Sanitize(req)
	if err := u.Validate.Struct(req); err != nil {
		return false, apierror.FromValidationError(err)
	}

	exists, err := u.UserRepo.ExistsByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to check if user (%s) exists: %v", req.Email, err)
		return false, apierror.InternalServerError
	}
	return exists, nil
}

// CreateUser creates a new user on Cognito (as well as in our database),
// and sends a verification code to the user's email address.
func (u *UserService) CreateUser(req *CreateUserRequest) apierror.ErrorResponse {
	utils.Sanitize(req)
	if err := u.Validate.Struct(req); err != nil {
		return apierror.FromValidationError(err)
	}

	found, err := u.UserRepo.ExistsByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to check if user already exists: %v", err)
		return apierror.InternalServerError
	}

	if found {
		return apierror.UserAlreadyExistsError
	}

	cogUser := &cognitoclient.User{Email: req.Email, Password: req.Password}
	uuid, apierr, revert := handleUserSignup(u.Cognito, cogUser)
	if apierr != nil {
		return apierr
	}

	// This is our user, in our database <3
	now := utils.NowUTC()
	user := &entity.User{
		SubUUID:       uuid,
		Username:      req.Username,
		Email:         req.Email,
		EmailVerified: false,
		IsAdmin:       false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = u.UserRepo.Save(user)
	if err != nil {
		// Well... for our case, I have no idea how can SQLite fail here,
		// but better safe than sorry?
		revert()
		log.Errorf("failed to create user: %v", err)
		return apierror.InternalServerError
	}
	return nil
}

func (u *UserService) Login(req *UserLoginRequest) (*UserLoginResponse, apierror.ErrorResponse) {
	if err := u.Validate.Struct(req); err != nil {
		return nil, apierror.FromValidationError(err)
	}

    user, err := u.UserRepo.FindByEmail(req.Email)
    if err != nil {
        log.Errorf("failed to fetch user from database: %v", err)
        return nil, apierror.InternalServerError
    }

    if user == nil {
        return nil, apierror.IDPUserNotFoundError
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

func (u *UserService) ConfirmSignup(req *ConfirmSignupRequest) apierror.ErrorResponse {
	if err := u.Validate.Struct(req); err != nil {
		return apierror.FromValidationError(err)
	}

	user, err := u.UserRepo.FindByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to fetch user from database: %v", err)
		return apierror.InternalServerError
	}

	if user == nil {
		return apierror.IDPUserNotFoundError
	}

	if user.EmailVerified {
		return apierror.UserAlreadyConfirmedError
	}

	confirms := &cognitoclient.UserConfirmation{
		Email: req.Email,
		Code:  req.Code,
	}

	apierr := handleSignupConfirmation(u.Cognito, confirms)
	if apierr != nil {
		return apierr
	}

	now := utils.NowUTC()
	user.EmailVerified = true
	user.UpdatedAt = now
	err = u.UserRepo.Save(user)
	if err != nil {
		log.Errorf("failed to update user (%d) verified status: %v", user.ID, err)
	}
	return nil
}

func (u *UserService) ResendConfirmation(req *ResendConfirmRequest) apierror.ErrorResponse {
	if err := u.Validate.Struct(req); err != nil {
		return apierror.FromValidationError(err)
	}

	user, err := u.UserRepo.FindByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to find user (%s) by email: %v", req.Email, err)
		return apierror.InternalServerError
	}

	if user == nil {
		return apierror.IDPUserNotFoundError
	}

	if user.EmailVerified {
		return apierror.UserAlreadyConfirmedError
	}

	apierr := handleConfirmResend(u.Cognito, req.Email)
	if apierr != nil {
		return apierr
	}

	// success
	user.EmailVerified = true
	user.UpdatedAt = utils.NowUTC()
	err = u.UserRepo.Save(user)
	if err != nil {
		log.Errorf("failed to set user (%s) as verified. INCONSISTENCY RISK: %v", req.Email, err)
		// No need to return an internal server error
	}
	return nil
}

func (u *UserService) fetchUser(rawId, sub string) (*entity.User, apierror.ErrorResponse) {
	if rawId == "@me" {
		return u.fetchBySub(sub)
	}
	return u.fetchByID(rawId)
}

func (u *UserService) fetchBySub(sub string) (*entity.User, apierror.ErrorResponse) {
	user, err := u.UserRepo.FindBySub(sub)
	if err != nil {
		log.Errorf("failed to find user (%s) by sub: %v", sub, err)
		return nil, apierror.InternalServerError
	}
	return user, nil
}

func (u *UserService) fetchByID(rawId string) (*entity.User, apierror.ErrorResponse) {
	userId, err := strconv.Atoi(rawId)
	if err != nil {
		return nil, apierror.NewInvalidParamTypeError("id", "int32")
	}
	user, err := u.UserRepo.FindByID(userId)
	if err != nil {
		log.Errorf("failed to find user (%s) by id: %v", rawId, err)
		return nil, apierror.InternalServerError
	}
	return user, nil
}

func handleUserSignup(cogClient cognitoclient.CognitoInterface, req *cognitoclient.User) (string, apierror.ErrorResponse, func()) {
	revert := func() {
		_ = cogClient.AdminDeleteUser(req.Email)
	}

	uuid, err := cogClient.SignUp(req)
	if err == nil {
		return uuid, nil, revert
	}

	switch {
	case errors.Is(err, &types.InvalidPasswordException{}):
		return "", apierror.IDPInvalidPasswordError, revert

	case errors.Is(err, &types.UsernameExistsException{}):
		return "", apierror.IDPExistingEmailError, revert

	default:
		log.Errorf("failed to signup user: %v", err)
		return "", apierror.InternalServerError, revert
	}
}

func handleUserSignin(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserLogin) (*cognitoclient.AuthCreate, apierror.ErrorResponse) {
	auth, err := cogClient.SignIn(req)
	if err == nil {
		return auth, nil
	}

	switch {
	case errors.Is(err, &types.UserNotFoundException{}):
		return nil, apierror.IDPUserNotFoundError

	case errors.Is(err, &types.UserNotConfirmedException{}):
		return nil, apierror.IDPUserNotConfirmedError

	case errors.Is(err, &types.NotAuthorizedException{}):
		return nil, apierror.IDPCredentialsMismatchError

	default:
		log.Errorf("failed to signin user (%s): %v", req.Email, err)
		return nil, apierror.InternalServerError
	}
}

func handleSignupConfirmation(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserConfirmation) apierror.ErrorResponse {
	err := cogClient.ConfirmAccount(req)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, &types.CodeMismatchException{}):
		return apierror.IDPConfirmCodeMismatchError

	case errors.Is(err, &types.ExpiredCodeException{}):
		return apierror.IDPConfirmCodeExpiredError

	case errors.Is(err, &types.UserNotFoundException{}):
		return apierror.IDPUserNotFoundError

	default:
		log.Errorf("failed to confirm user (%s): %v", req.Email, err)
		return apierror.InternalServerError
	}
}

func handleConfirmResend(cogClient cognitoclient.CognitoInterface, email string) apierror.ErrorResponse {
	err := cogClient.ResendConfirmation(email)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, &types.UserNotFoundException{}):
		return apierror.IDPUserNotFoundError

	case errors.Is(err, &types.InvalidParameterException{}):
		return apierror.IDPInvalidParameterError

	default:
		log.Errorf("failed to resend confirmation code to email (%s): %v", email, err)
		return apierror.InternalServerError
	}
}

func toUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		IsAdmin:   user.IsAdmin,
		CreatedAt: utils.FormatEpoch(user.CreatedAt),
		UpdatedAt: utils.FormatEpoch(user.UpdatedAt),
	}
}
