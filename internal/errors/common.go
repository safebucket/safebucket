package apierrors

import "errors"

type APIError struct {
	Status int
	Code   string
}

func New(status int, code string) *APIError {
	return &APIError{Status: status, Code: code}
}

func (e *APIError) Error() string {
	return e.Code
}

func OrNew(err error, status int, code string) error {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr
	}
	return New(status, code)
}
