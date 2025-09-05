package apierror

import (
	"fmt"
	"net/http"
)

type APIError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

var (
	MalformedJSONError  = NewError(400, "Malformed JSON body")
	InternalServerError = NewError(500, "Internal server error")

	NotFoundError       = NewError(404, "Resource not found")
	DuplicateAliasError = NewError(400, "Cannot have duplicate aliases")
	InvalidIDError      = NewError(400, "The provided ID is invalid, IDs are usually int32 > 0")

	/*
	 * Used for authentications
	 */
	UserAlreadyConfirmedError   = NewError(400, "User is already confirmed")
	IDPInvalidPasswordError     = NewError(400, "Provided password does not meet requirements")
	IDPExistingEmailError       = NewError(400, "Email already exists")
	IDPUserNotFoundError        = NewError(404, "User not found")
	IDPUserNotConfirmedError    = NewError(400, "User is not confirmed yet")
	IDPCredentialsMismatchError = NewError(400, "Credentials mismatch")
	IDPConfirmCodeMismatchError = NewError(400, "Confirmation code mismatch")
	IDPConfirmCodeExpiredError  = NewError(400, "Confirmation code has expired")
	IDPInvalidParameterError    = NewError(400, "Invalid parameters provided, the user is likely already verified")
)

func NewError(status int, msg string, args ...any) *APIError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &APIError{Status: status, Message: msg}
}

func NewAliasLengthError(alias string, min, max int) *APIError {
	return NewError(http.StatusBadRequest, "Notes aliases must be in range of [%d - %d], provided (%d): %s",
		min, max, len(alias), alias)
}

func NewInvalidParamTypeError(name, dataType string) *APIError {
	return NewError(http.StatusBadRequest, "Parameter '%s' has invalid type, expected: %s", name, dataType)
}
