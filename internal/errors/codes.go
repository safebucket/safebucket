package apierrors

const (
	CodeForbidden           = "FORBIDDEN"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeSessionRevoked      = "SESSION_REVOKED"
	CodeBadRequest          = "BAD_REQUEST"
	CodeInvalidUUID         = "INVALID_UUID"
)

// HTTP 400 Bad Request.
const (
	CodeCannotDownloadTrashed = "CANNOT_DOWNLOAD_TRASHED_FILE"
	CodeFolderNameConflict    = "FOLDER_NAME_CONFLICT"
)

// HTTP 403 Forbidden.
const (
	CodeFileExpired = "FILE_EXPIRED"
)

// HTTP 410 Gone.
const (
	CodeFolderTrashExpired = "FOLDER_TRASH_EXPIRED"
)
