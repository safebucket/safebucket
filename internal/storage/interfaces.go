package storage

import "mime"

const (
	bucketsPrefix = "buckets/"
	trashPrefix   = "trash/"
	folderPath    = "folders"
	filePath      = "files"

	respContentDisposition = "response-content-disposition"
	respContentType        = "response-content-type"
)

type GetObjectOptions struct {
	InlineContentType string
	DownloadFilename  string
}

type UploadTarget struct {
	URL      string
	Method   string
	FormData map[string]string
}

func attachmentDisposition(filename string) string {
	if filename == "" {
		return "attachment"
	}
	if disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename}); disposition != "" {
		return disposition
	}
	return "attachment"
}

type IStorage interface {
	PresignedGetObject(objectPath string, opts GetObjectOptions) (string, error)
	PresignedUpload(
		path string,
		size int,
		metadata map[string]string,
	) (UploadTarget, error)
	StatObject(path string) (map[string]string, error)
	ListObjects(prefix string, maxKeys int32) ([]string, error)
	RemoveObject(path string) error
	RemoveObjects(paths []string) error
	EnsureTrashLifecyclePolicy(retentionDays int) error
	MarkAsTrashed(objectPath string, model interface{}) error
	UnmarkAsTrashed(objectPath string, model interface{}) error
	IsTrashMarkerPath(path string) (isMarker bool, originalPath string)
	GetBucketName() string
}
