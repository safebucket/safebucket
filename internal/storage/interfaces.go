package storage

import (
	"mime"
	"time"

	"github.com/safebucket/safebucket/internal/models"
)

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

type PartInfo struct {
	PartNumber   int
	Size         int64
	ETag         string
	LastModified time.Time
}

type PresignedUpload struct {
	Response models.FileUploadResponse
	UploadID string
	PartSize int64
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
	PresignUpload(objectPath string, size int, metadata map[string]string) (PresignedUpload, error)
	SupportsMultipart() bool
	ListObjectParts(path, uploadID string) ([]PartInfo, error)
	CompleteMultipartUpload(path, uploadID string, parts []PartInfo, metadata map[string]string) error
	AbortMultipartUpload(path, uploadID string) error
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
