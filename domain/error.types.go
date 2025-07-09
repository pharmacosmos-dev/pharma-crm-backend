package domain

import (
	"net/http"

	"github.com/pharma-crm-backend/pkg/utils"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// region error types
var (
	BadRequestError     = NewError(http.StatusBadRequest, "bad.request")
	InternalServerError = NewError(http.StatusInternalServerError, "internal.server.error")
	InvalidBodyError    = NewError(http.StatusBadRequest, "invalid.request.body")
	UnauthorizedError   = NewError(http.StatusUnauthorized, "unauthorized")
	ForbiddinError      = NewError(http.StatusForbidden, "forbidden")
	InvalidQueryError   = NewError(http.StatusBadRequest, "invalid.query.param")
)

// Error makes it compatible with the `error` interface.
func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new Error instance with an optional message
func NewError(code int, message ...string) *Error {
	err := &Error{
		Code:    code,
		Message: utils.StatusMessage(code),
	}
	if len(message) > 0 {
		err.Message = message[0]
	}
	return err
}
