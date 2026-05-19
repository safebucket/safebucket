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
