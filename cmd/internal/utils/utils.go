package utils

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/labstack/gommon/log"
	"path/filepath"
	"reflect"
	"simplenotes/cmd/internal/utils/apierror"
	"slices"
	"strings"
	"time"
)

var (
	invalidPwd    *types.InvalidPasswordException
	userExists    *types.UsernameExistsException
	userNotFound  *types.UserNotFoundException
	notConfirmed  *types.UserNotConfirmedException
	notAuthorized *types.NotAuthorizedException
	codeMismatch  *types.CodeMismatchException
	expiredCode   *types.ExpiredCodeException
	invalidParam  *types.InvalidParameterException
)

func FormatEpoch(millis int64) string {
	return time.UnixMilli(millis).
		UTC().
		Format(time.RFC3339)
}

func NowUTC() int64 {
	return time.Now().
		UTC().
		UnixMilli()
}

// IsEmpty reports whether all pointer fields in the given struct are nil.
// This is useful for PATCH handlers that rely on pointer fields to
// distinguish provided fields from omitted ones. If every pointer field
// is nil after binding, the payload is effectively empty.
//
// v may be a struct or a pointer to a struct. Non-pointer fields
// (e.g., slices or value types) are ignored.
func IsEmpty(v any) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)

		if f.Kind() == reflect.Ptr && !f.IsNil() {
			return false
		}
	}
	return true
}

// CheckFileExt reports whether the file's extension is included in the
// provided list of valid extensions. Extensions in the valid slice must not
// include a leading dot (e.g., ["txt", "pdf"]).
//
// It returns the extension (without the leading dot) and a boolean
// indicating whether the extension is accepted. If the file has no
// extension, the returned extension is an empty string and the validity
// flag is false.
func CheckFileExt(fileName string, valid []string) (string, bool) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		return "", false
	}
	return ext, slices.Contains(valid, ext[1:])
}

func MapCognitoError(err error) apierror.ErrorResponse {
	switch {
	case errors.As(err, &invalidPwd):
		return apierror.IDPInvalidPasswordError
	case errors.As(err, &userExists):
		return apierror.IDPExistingEmailError
	case errors.As(err, &userNotFound):
		return apierror.IDPUserNotFoundError
	case errors.As(err, &notConfirmed):
		return apierror.IDPUserNotConfirmedError
	case errors.As(err, &notAuthorized):
		return apierror.IDPCredentialsMismatchError
	case errors.As(err, &codeMismatch):
		return apierror.IDPConfirmCodeMismatchError
	case errors.As(err, &expiredCode):
		return apierror.IDPConfirmCodeExpiredError
	case errors.As(err, &invalidParam):
		return apierror.IDPInvalidParameterError
	default:
		// Log the original underlying error for debugging purposes
		log.Errorf("unmapped cognito error: %v", err)
		return apierror.InternalServerError
	}
}

func Sanitize(o any) {
	v := reflect.ValueOf(o)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("sanitize: expected pointer to struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		panic("sanitize: expected struct")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString(sanitizeString(field.String()))

		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					field.Index(j).SetString(sanitizeString(field.Index(j).String()))
				}
			}
		}
	}
}

func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}
