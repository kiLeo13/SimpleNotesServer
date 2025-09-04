package validators

import (
	"github.com/go-playground/validator/v10"
	"regexp"
	"unicode"
)

const (
	PasswordMinLength = 8
	PasswordMaxLength = 64
)

var specialRegex = regexp.MustCompile(`[\\^$*.\[\]{}()?"!@#%&/\\,><':;|_~` + "`" + `=+\-]`)

func PasswordValidator(fl validator.FieldLevel) bool {
	password, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	length := len(password)
	if length < PasswordMinLength || length > PasswordMaxLength {
		return false
	}

	hasSpecial := specialRegex.MatchString(password)
	var hasUpper, hasLower, hasDigit bool

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true

		case unicode.IsLower(ch):
			hasLower = true

		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}
