package service

import (
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/policy"
	cognitoclient "simplenotes/cmd/internal/infrastructure/aws/cognito"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
)

type EmailStatus string

const (
	// EmailStatusAvailable indicates that the email is available for registration.
	EmailStatusAvailable EmailStatus = "AVAILABLE"
	// EmailStatusExists indicates that the email is already in use by some user.
	EmailStatusExists EmailStatus = "TAKEN"
	// EmailStatusVerifying indicates that the email is in the process of verification.
	EmailStatusVerifying EmailStatus = "VERIFYING"
)

type UserRepository interface {
	FindAllActive() ([]*entity.User, error)
	FindActiveBySub(sub string) (*entity.User, error)
	FindActiveByEmail(email string) (*entity.User, error)
	FindActiveByID(id int) (*entity.User, error)
	FindByID(id int) (*entity.User, error)
	SoftDelete(user *entity.User) error
	ExistsActiveByEmail(email string) (bool, error)
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

type UpdateUserRequest struct {
	Username  *string `json:"username" validate:"omitempty,min=2,max=80"`
	Perms     *int64  `json:"perms" validate:"omitempty,min=0"`
	Suspended *bool   `json:"suspended" validate:"omitempty"`
}

type ConfirmSignupRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,min=1,max=8"`
}

type ResendConfirmRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UserStatusRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UserResponse struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Perms      int64  `json:"permissions"`
	IsVerified *bool  `json:"is_verified,omitempty"` // Req (Manage Users)
	Suspended  *bool  `json:"suspended,omitempty"`   // Req (Manage Users | Punish Users)
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type UserLoginResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type UserService struct {
	UserRepo   UserRepository
	Validate   *validator.Validate
	Cognito    cognitoclient.CognitoInterface
	UserPolicy *policy.UserPolicy
}

func NewUserService(userRepo UserRepository, validate *validator.Validate, cogClient cognitoclient.CognitoInterface, userPolicy *policy.UserPolicy) *UserService {
	return &UserService{
		UserRepo:   userRepo,
		Validate:   validate,
		Cognito:    cogClient,
		UserPolicy: userPolicy,
	}
}

func (u *UserService) GetUsers(actor *entity.User) ([]*UserResponse, apierror.ErrorResponse) {
	users, err := u.UserRepo.FindAllActive()
	if err != nil {
		return nil, nil
	}

	resp := make([]*UserResponse, len(users))
	for i, user := range users {
		resp[i] = toUserResponse(user, actor)
	}
	return resp, nil
}

func (u *UserService) GetUser(actor *entity.User, rawId string) (*UserResponse, apierror.ErrorResponse) {
	user, apierr := u.fetchUser(actor, rawId, true)
	if apierr != nil {
		return nil, apierr
	}

	if user == nil {
		return nil, apierror.NotFoundError
	}

	resp := toUserResponse(user, actor)
	return resp, nil
}

func (u *UserService) UpdateUser(actor *entity.User, targetId string, req *UpdateUserRequest) (*UserResponse, apierror.ErrorResponse) {
	utils.Sanitize(req)
	if err := u.Validate.Struct(req); err != nil {
		return nil, apierror.FromValidationError(err)
	}

	target, apierr := u.fetchByID(targetId, false)
	if apierr != nil {
		return nil, apierr
	}

	updater := &userUpdater{
		actor:  actor,
		target: target,
		policy: u.UserPolicy,
	}

	updater.setProfileString(req.Username, &target.Username)
	updater.setPermissions(req.Perms)
	updater.setSuspended(req.Suspended)

	if updater.err != nil {
		return nil, updater.err
	}

	if updater.dirty {
		target.UpdatedAt = utils.NowUTC()
		if err := u.UserRepo.Save(target); err != nil {
			log.Errorf("actor %s failed to update user %s: %v", actor.SubUUID, targetId, err)
			return nil, apierror.InternalServerError
		}
	}
	return toUserResponse(target, actor), nil
}

func (u *UserService) DeleteUser(actor *entity.User, targetRawID string) apierror.ErrorResponse {
	target, err := u.fetchByID(targetRawID, false)
	if err != nil {
		log.Errorf("failed to fetch user by ID %s: %v", targetRawID, err)
		return apierror.InternalServerError
	}

	if target == nil {
		return apierror.NotFoundError
	}

	if perr := u.UserPolicy.CanDeleteUser(actor, target); perr != nil {
		return perr
	}

	if derr := u.UserRepo.SoftDelete(target); derr != nil {
		log.Errorf("failed to delete user %d: %v", target.ID, derr)
		return apierror.InternalServerError
	}
	return nil
}

func (u *UserService) CheckEmail(req *UserStatusRequest) (*EmailStatus, apierror.ErrorResponse) {
	utils.Sanitize(req)
	if err := u.Validate.Struct(req); err != nil {
		return nil, apierror.FromValidationError(err)
	}

	var status EmailStatus
	user, err := u.UserRepo.FindActiveByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to check if user (%s) exists: %v", req.Email, err)
		return nil, apierror.InternalServerError
	}

	switch {
	case user == nil:
		status = EmailStatusAvailable
	case !user.EmailVerified:
		status = EmailStatusVerifying
	default:
		status = EmailStatusExists
	}
	return &status, nil
}

// CreateUser creates a new user on Cognito (as well as in our database),
// and sends a verification code to the user's email address.
func (u *UserService) CreateUser(req *CreateUserRequest) apierror.ErrorResponse {
	utils.Sanitize(req)
	if err := u.Validate.Struct(req); err != nil {
		return apierror.FromValidationError(err)
	}

	found, err := u.UserRepo.ExistsActiveByEmail(req.Email)
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

	user, err := u.UserRepo.FindActiveByEmail(req.Email)
	if err != nil {
		log.Errorf("failed to fetch user from database: %v", err)
		return nil, apierror.InternalServerError
	}

	if user == nil {
		return nil, apierror.IDPUserNotFoundError
	}

	if user.Suspended {
		return nil, apierror.MissingAccessError
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

	user, err := u.UserRepo.FindActiveByEmail(req.Email)
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

	user, err := u.UserRepo.FindActiveByEmail(req.Email)
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

// fetchUser tries to resolve the params into a real user.
//
// When 'force' is 'true', even deleted users can be returned.
func (u *UserService) fetchUser(requester *entity.User, rawId string, force bool) (*entity.User, apierror.ErrorResponse) {
	if rawId == "@me" {
		return requester, nil
	}
	return u.fetchByID(rawId, force)
}

func (u *UserService) fetchBySub(sub string) (*entity.User, apierror.ErrorResponse) {
	user, err := u.UserRepo.FindActiveBySub(sub)
	if err != nil {
		log.Errorf("failed to find user (%s) by sub: %v", sub, err)
		return nil, apierror.InternalServerError
	}
	return user, nil
}

func (u *UserService) fetchByID(rawId string, force bool) (*entity.User, apierror.ErrorResponse) {
	userId, err := strconv.Atoi(rawId)
	if err != nil {
		return nil, apierror.NewInvalidParamTypeError("id", "int32")
	}

	var user *entity.User
	if force {
		user, err = u.UserRepo.FindByID(userId)
	} else {
		user, err = u.UserRepo.FindActiveByID(userId)
	}

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
	if err != nil {
		return "", utils.MapCognitoError(err), revert
	}
	return uuid, nil, revert
}

func handleUserSignin(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserLogin) (*cognitoclient.AuthCreate, apierror.ErrorResponse) {
	auth, err := cogClient.SignIn(req)
	if err != nil {
		return nil, utils.MapCognitoError(err)
	}
	return auth, nil
}

func handleSignupConfirmation(cogClient cognitoclient.CognitoInterface, req *cognitoclient.UserConfirmation) apierror.ErrorResponse {
	err := cogClient.ConfirmAccount(req)
	if err != nil {
		return utils.MapCognitoError(err)
	}
	return nil
}

func handleConfirmResend(cogClient cognitoclient.CognitoInterface, email string) apierror.ErrorResponse {
	err := cogClient.ResendConfirmation(email)
	if err != nil {
		return utils.MapCognitoError(err)
	}
	return nil
}

func toUserResponse(user, requester *entity.User) *UserResponse {
	if !user.Active {
		return toDeletedUserResponse(user)
	}

	resp := &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Perms:     int64(user.Permissions),
		CreatedAt: utils.FormatEpoch(user.CreatedAt),
		UpdatedAt: utils.FormatEpoch(user.UpdatedAt),
	}

	hasMngUsers := requester.Permissions.HasEffective(entity.PermissionManageUsers)
	hasPunishUsers := requester.Permissions.HasEffective(entity.PermissionPunishUsers)
	if hasMngUsers {
		resp.IsVerified = &user.EmailVerified
	}

	if hasPunishUsers || hasMngUsers {
		resp.Suspended = &user.Suspended
	}
	return resp
}

func toDeletedUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  "Deleted User",
		Perms:     0,
		CreatedAt: utils.FormatEpoch(0),
		UpdatedAt: utils.FormatEpoch(0),
	}
}
