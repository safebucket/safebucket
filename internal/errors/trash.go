package apierrors

// HTTP 400 Bad Request.
const (
	ErrCannotDownloadTrashed = "CANNOT_DOWNLOAD_TRASHED_FILE"
	ErrFolderNameConflict    = "FOLDER_NAME_CONFLICT"
)

// HTTP 403 Forbidden.
const (
	ErrFileExpired = "FILE_EXPIRED"
)

// HTTP 410 Gone.
const (
	ErrFolderTrashExpired = "FOLDER_TRASH_EXPIRED"
)
