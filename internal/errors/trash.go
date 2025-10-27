package errors

// Trash error codes - HTTP 400 Bad Request.
const (
	ErrFileAlreadyTrashed    = "FILE_ALREADY_TRASHED"
	ErrFileNotInTrash        = "FILE_NOT_IN_TRASH"
	ErrFolderAlreadyTrashed  = "FOLDER_ALREADY_TRASHED"
	ErrFolderNotInTrash      = "FOLDER_NOT_IN_TRASH"
	ErrCannotDownloadTrashed = "CANNOT_DOWNLOAD_TRASHED_FILE"
	ErrNotAFolder            = "NOT_A_FOLDER"
	ErrFolderNameConflict    = "FOLDER_NAME_CONFLICT"
)

// Trash error codes - HTTP 410 Gone.
const (
	ErrFileTrashExpired   = "FILE_TRASH_EXPIRED"
	ErrFolderTrashExpired = "FOLDER_TRASH_EXPIRED"
)
