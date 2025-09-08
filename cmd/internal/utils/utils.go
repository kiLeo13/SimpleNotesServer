package utils

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
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

func CheckFileExt(fileName string, valid []string) (string, bool) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		return "", false
	}
	return ext, slices.Contains(valid, ext[1:])
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
