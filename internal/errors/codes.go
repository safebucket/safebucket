package apierrors

const (
	CodeForbidden           = "FORBIDDEN"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeSessionRevoked      = "SESSION_REVOKED"
	CodeBadRequest          = "BAD_REQUEST"
	CodeInvalidUUID         = "INVALID_UUID"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS" //nolint:gosec // error code name, not a secret.
	CodeInvalidPassword     = "INVALID_PASSWORD"
	CodeUserNotFound        = "USER_NOT_FOUND"
	CodeProviderNotFound    = "PROVIDER_NOT_FOUND"
	CodeInvalidProviderName = "INVALID_PROVIDER_NAME"
)

// HTTP 400 Bad Request.
const (
	CodeCannotDownloadTrashed = "CANNOT_DOWNLOAD_TRASHED_FILE"
	CodeFolderNameConflict    = "FOLDER_NAME_CONFLICT"
)

// HTTP 403 Forbidden.
const (
	CodeFileExpired              = "FILE_EXPIRED"
	CodePasswordChangeNotAllowed = "PASSWORD_CHANGE_NOT_ALLOWED"
)

// HTTP 410 Gone.
const (
	CodeFolderTrashExpired = "FOLDER_TRASH_EXPIRED"
)

// HTTP 503 Service Unavailable.
const (
	CodeAuthProviderUnavailable = "AUTH_PROVIDER_UNAVAILABLE"
)

// OIDC / OAuth provider flow.
const (
	CodeOAuthExchangeFailed = "OAUTH_EXCHANGE_FAILED"
	CodeOAuthUserinfoFailed = "OAUTH_USERINFO_FAILED"
	CodeIDTokenMissing      = "ID_TOKEN_MISSING"
	CodeIDTokenVerifyFailed = "ID_TOKEN_VERIFY_FAILED" //nolint:gosec // error code name, not a secret.
	CodeOIDCStateNotFound   = "OIDC_STATE_NOT_FOUND"
	CodeOIDCStateMismatch   = "OIDC_STATE_MISMATCH"
	CodeOIDCNonceNotFound   = "OIDC_NONCE_NOT_FOUND"
	CodeOIDCNonceMismatch   = "OIDC_NONCE_MISMATCH"
)
