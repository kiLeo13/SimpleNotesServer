package contract

type EmailStatus string

const (
	EmailStatusAvailable EmailStatus = "AVAILABLE"
	EmailStatusExists    EmailStatus = "TAKEN"
	EmailStatusVerifying EmailStatus = "VERIFYING"
)

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=2,max=80"`
	Email    string `json:"email" validate:"required,email"`
	// Custom validators like 'hasspecial' usually work via the Validator engine,
	// so just keeping the tag string here is fine.
	Password string `json:"password" validate:"required,min=8,max=64,hasspecial,hasdigit,hasupper,haslower"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type UpdateUserRequest struct {
	Username  *string `json:"username" validate:"omitempty,min=2,max=80"`
	Perms     *int64  `json:"permissions" validate:"omitempty,min=0"`
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
	IsVerified *bool  `json:"is_verified,omitempty"`
	Suspended  *bool  `json:"suspended,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type UserLoginResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}
