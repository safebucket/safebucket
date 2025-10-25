package storage

type IStorage interface {
	PresignedGetObject(path string) (string, error)
	PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error)
	StatObject(path string) (map[string]string, error)
	ListObjects(prefix string, maxKeys int32) ([]string, error)
	RemoveObject(path string) error
	RemoveObjects(paths []string) error
	SetObjectTags(path string, tags map[string]string) error
	GetObjectTags(path string) (map[string]string, error)
	RemoveObjectTags(path string, tagsToRemove []string) error
}
