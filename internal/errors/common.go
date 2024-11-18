package errors

import (
	"errors"
)

var (
	CreateFailed = errors.New("CREATE_FAILED")
)

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}
