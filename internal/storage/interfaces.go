package storage

type IStorage interface {
	PresignedGetObject(path string) (string, error)
	PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error)
	StatObject(path string) error
	RemoveObject(path string) error
}
