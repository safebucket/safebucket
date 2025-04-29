package errors

import "errors"

var (
	GenerateAccessTokenFailed  = errors.New("GENERATE_ACCESS_TOKEN_FAILED")
	GenerateRefreshTokenFailed = errors.New("GENERATE_REFRESH_TOKEN_FAILED")
)
