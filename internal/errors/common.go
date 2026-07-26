package apierrors

import "errors"

var ErrMultipartNotSupported = errors.New("multipart upload not supported by storage provider")

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
