package errors

import (
	"errors"
)

var (
	ErrInternalServer = errors.New("INTERNAL_SERVER_ERROR")
	ErrCreateFailed   = errors.New("CREATE_FAILED")
	ErrDeleteFailed   = errors.New("DELETE_FAILED")
)

type APIError struct {
	Code    int
	Message string
}

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func (e *APIError) Error() string {
	return e.Message
}
