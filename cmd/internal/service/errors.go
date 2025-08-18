package service

import (
	"fmt"
	"net/http"
)

type APIError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

var (
	ErrorMalformedJSON  = NewError(400, "Malformed JSON body")
	ErrorInternalServer = NewError(500, "Internal server error")

	ErrorNotFound = NewError(404, "Resource not found")

	ErrorDuplicateAlias = NewError(400, "Cannot have duplicate aliases")
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
