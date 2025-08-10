package errors

import "errors"

var (
	ErrorGenerateAccessTokenFailed  = errors.New("GENERATE_ACCESS_TOKEN_FAILED")
	ErrorGenerateRefreshTokenFailed = errors.New("GENERATE_REFRESH_TOKEN_FAILED")
)
