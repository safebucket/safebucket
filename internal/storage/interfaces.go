package storage

const (
	bucketsPrefix = "buckets/"
	trashPrefix   = "trash/"
	folderPath    = "folders"
	filePath      = "files"
)

type IStorage interface {
	PresignedGetObject(path string) (string, error)
	PresignedPostPolicy(
		path string,
		size int,
		metadata map[string]string,
	) (string, map[string]string, error)
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
