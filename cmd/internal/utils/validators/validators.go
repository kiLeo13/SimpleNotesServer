package validators

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
	"reflect"
	"regexp"
	"unicode"
)

var (
	specialRegex = regexp.MustCompile(`[\\^$*.\[\]{}()?"!@#%&/\\,><':;|_~` + "`" + `=+\-]`)
	hasSpaces    = regexp.MustCompile(`\s+`)
)

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

// NoWhiteSpaces returns false if the string contains any whitespace (rejecting the user input).
func NoWhiteSpaces(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}

	str := field.String()
	return !hasSpaces.MatchString(str)
}

func NoDupes(fl validator.FieldLevel) bool {
	slice := fl.Field()
	if slice.Kind() != reflect.Slice {
		log.Warnf("validator 'nodupes' applied to non-slice type: %s\n", slice.Kind().String())
		return false
	}

	length := slice.Len()
	seen := make(map[any]bool, length)
	for i := 0; i < length; i++ {
		val := slice.Index(i).Interface()
		if _, exists := seen[val]; exists {
			return false
		}
		seen[val] = true
	}
	return true
}
