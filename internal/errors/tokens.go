package errors

import "errors"

var (
	ErrGenerateAccessTokenFailed  = errors.New("GENERATE_ACCESS_TOKEN_FAILED")
	ErrGenerateRefreshTokenFailed = errors.New("GENERATE_REFRESH_TOKEN_FAILED")
)
