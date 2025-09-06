package validators

import (
	"github.com/go-playground/validator/v10"
	"regexp"
	"unicode"
)

var specialRegex = regexp.MustCompile(`[\\^$*.\[\]{}()?"!@#%&/\\,><':;|_~` + "`" + `=+\-]`)

func HasUpper(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	for _, ch := range val {
		if unicode.IsUpper(ch) {
			return true
		}
	}
	return false
}

func HasLower(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	for _, ch := range val {
		if unicode.IsLower(ch) {
			return true
		}
	}
	return false
}

func HasDigit(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	for _, ch := range val {
		if unicode.IsDigit(ch) {
			return true
		}
	}
	return false
}

func HasSpecial(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	return specialRegex.MatchString(val)
}
