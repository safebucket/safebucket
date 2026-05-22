package apierrors

// Authentication and session codes.
//
//nolint:gosec // G101 false positives; these are response codes sent to clients.
const (
	CodeSessionRevoked             = "SESSION_REVOKED"
	CodeSessionNotFound            = "SESSION_NOT_FOUND"
	CodeInvalidCredentials         = "INVALID_CREDENTIALS"
	CodeIncorrectPassword          = "INCORRECT_PASSWORD"
	CodeInvalidPassword            = "INVALID_PASSWORD"
	CodeInvalidAccessToken         = "INVALID_ACCESS_TOKEN"
	CodeInvalidAccessTokenAudience = "INVALID_ACCESS_TOKEN_AUDIENCE"
	CodeTokenGenerationFailed      = "TOKEN_GENERATION_FAILED"
	CodeGenerateAccessTokenFailed  = "GENERATE_ACCESS_TOKEN_FAILED"
	CodeGenerateRefreshTokenFailed = "GENERATE_REFRESH_TOKEN_FAILED"
)

// Identity provider codes.
//
//nolint:gosec // G101 false positives; these are response codes, not credentials.
const (
	CodeInvalidProviderName = "INVALID_PROVIDER_NAME"
	CodeProviderNotFound    = "PROVIDER_NOT_FOUND"
	CodeUnknownUserProvider = "UNKNOWN_USER_PROVIDER"
	CodeOAuthExchangeFailed = "OAUTH_EXCHANGE_FAILED"
	CodeOAuthUserinfoFailed = "OAUTH_USERINFO_FAILED"
	CodeIDTokenMissing      = "ID_TOKEN_MISSING"
	CodeIDTokenVerifyFailed = "ID_TOKEN_VERIFY_FAILED"
	CodeOIDCStateNotFound   = "OIDC_STATE_NOT_FOUND"
	CodeOIDCStateMismatch   = "OIDC_STATE_MISMATCH"
	CodeOIDCNonceNotFound   = "OIDC_NONCE_NOT_FOUND"
	CodeOIDCNonceMismatch   = "OIDC_NONCE_MISMATCH"
)

const (
	CodeChallengeNotFound = "CHALLENGE_NOT_FOUND"
	CodeChallengeExpired  = "CHALLENGE_EXPIRED"
	CodeChallengeLocked   = "CHALLENGE_LOCKED"
	CodeWrongCode         = "WRONG_CODE"
)

const (
	CodeUserNotFound      = "USER_NOT_FOUND"
	CodeUserAlreadyExists = "USER_ALREADY_EXISTS"
)
