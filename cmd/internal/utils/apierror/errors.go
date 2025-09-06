package apierror

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"
)

// ErrorResponse abstracts all API error responses to the user.
//
// This interface does not implement `error`, since its only purpose
// is to be used for API responses and not for logging circumstances.
//
// In general, the whole ErrorResponse can be sent for serialization.
type ErrorResponse interface {
	// Code is the HTTP status code to be returned.
	Code() int
}

type APIError struct {
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (a *APIError) Code() int {
	return a.Status
}

type StructuredError struct {
	Errors map[string][]string `json:"errors"`
	Status int                 `json:"-"`
}

func (s *StructuredError) Code() int {
	return s.Status
}

func (s *StructuredError) Add(field, problem string) {
	s.Errors[field] = append(s.Errors[field], problem)
}

var (
	MalformedJSONError  = NewSimple(400, "Malformed JSON body")
	InternalServerError = NewSimple(500, "Internal server error")

	NotFoundError       = NewSimple(404, "Resource not found")
	DuplicateAliasError = NewSimple(400, "Cannot have duplicate aliases")
	InvalidIDError      = NewSimple(400, "The provided ID is invalid, IDs are usually int32 > 0")

	/*
	 * Used for authentications
	 */
	UserAlreadyConfirmedError   = NewSimple(400, "User is already confirmed")
	IDPInvalidPasswordError     = NewSimple(400, "Provided password does not meet requirements")
	IDPExistingEmailError       = NewSimple(400, "Email already exists")
	IDPUserNotFoundError        = NewSimple(404, "User not found")
	IDPUserNotConfirmedError    = NewSimple(400, "User is not confirmed yet")
	IDPCredentialsMismatchError = NewSimple(400, "Credentials mismatch")
	IDPConfirmCodeMismatchError = NewSimple(400, "Confirmation code mismatch")
	IDPConfirmCodeExpiredError  = NewSimple(400, "Confirmation code has expired")
	IDPInvalidParameterError    = NewSimple(400, "Invalid parameters provided, the user is likely already verified")
)

func FromValidationError(err error) *StructuredError {
	var ve validator.ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		return nil
	}

	problems := map[string][]string{}
	for _, fe := range err.(validator.ValidationErrors) {
		field := strings.ToLower(fe.Field())

		switch fe.Tag() {
		case "required":
			problems[field] = append(problems[field], "This field is required")
		case "min":
			problems[field] = append(problems[field], "Value is too short, min: "+fe.Param())
		case "max":
			problems[field] = append(problems[field], "Value is too long, max: "+fe.Param())
		case "hasupper":
			problems[field] = append(problems[field], "Value must have at least one uppercase character")
		case "haslower":
			problems[field] = append(problems[field], "Value must have at least one lowercase character")
		case "hasdigit":
			problems[field] = append(problems[field], "Value must have at least one number")
		case "hasspecial":
			problems[field] = append(problems[field], "Value must have at least one special character")
		case "email":
			problems[field] = append(problems[field], "Value must be a valid email address")

		default:
			problems[field] = append(problems[field], "Invalid value provided")
		}
	}

	return &StructuredError{
		Errors: problems,
		Status: http.StatusBadRequest,
	}
}

func NewSimple(status int, msg string, args ...any) *APIError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &APIError{Status: status, Message: msg}
}

func NewStructured(code int) *StructuredError {
	return &StructuredError{
		Errors: make(map[string][]string),
		Status: code,
	}
}

func NewAliasLengthError(alias string, min, max int) *APIError {
	return NewSimple(http.StatusBadRequest, "Notes aliases must be in range of [%d - %d], provided (%d): %s",
		min, max, len(alias), alias)
}

func NewInvalidParamTypeError(name, dataType string) *APIError {
	return NewSimple(http.StatusBadRequest, "Parameter '%s' has invalid type, expected: %s", name, dataType)
}
