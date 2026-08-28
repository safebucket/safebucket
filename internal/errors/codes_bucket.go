package apierrors

const (
	CodeBucketNotFound = "BUCKET_NOT_FOUND"
)

const (
	CodeSharingDisabledForProvider = "SHARING_DISABLED_FOR_PROVIDER"
)

const (
	CodeFileNotFound                = "FILE_NOT_FOUND"
	CodeFileAlreadyExists           = "FILE_ALREADY_EXISTS"
	CodeFileNotInTrash              = "FILE_NOT_IN_TRASH"
	CodeFileNotInStorage            = "FILE_NOT_IN_STORAGE"
	CodeFileNameConflict            = "FILE_NAME_CONFLICT"
	CodeFileRestoreInProgress       = "FILE_RESTORE_IN_PROGRESS"
	CodeFileExpired                 = "FILE_EXPIRED"
	CodeCannotDownloadTrashed       = "CANNOT_DOWNLOAD_TRASHED_FILE"
	CodeInvalidFileStatusTransition = "INVALID_FILE_STATUS_TRANSITION"
	CodeInvalidStatus               = "INVALID_STATUS"
	CodeMaxUploadsReached           = "MAX_UPLOADS_REACHED"
	CodeMultipartSizeMismatch       = "MULTIPART_SIZE_MISMATCH"
	CodeMultipartCompleteFailed     = "MULTIPART_COMPLETE_FAILED"
)

const (
	CodeFolderNotFound           = "FOLDER_NOT_FOUND"
	CodeFolderAlreadyExists      = "FOLDER_ALREADY_EXISTS"
	CodeFolderNotInTrash         = "FOLDER_NOT_IN_TRASH"
	CodeFolderRestoreInProgress  = "FOLDER_RESTORE_IN_PROGRESS"
	CodeFolderNameConflict       = "FOLDER_NAME_CONFLICT"
	CodeParentFolderNameConflict = "PARENT_FOLDER_NAME_CONFLICT"
	CodeParentFolderNotFound     = "PARENT_FOLDER_NOT_FOUND"
	CodeFolderTrashExpired       = "FOLDER_TRASH_EXPIRED"
)

const (
	CodeItemAlreadyTrashed = "ITEM_ALREADY_TRASHED"
	CodeItemNotTrashable   = "ITEM_NOT_TRASHABLE"
)
