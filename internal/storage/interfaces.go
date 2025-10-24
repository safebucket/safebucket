package storage

import "time"

type IStorage interface {
	PresignedGetObject(path string) (string, error)
	PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error)
	StatObject(path string) (map[string]string, error)
	ListObjects(prefix string, maxKeys int32) ([]string, error)
	RemoveObject(path string) error
	RemoveObjects(paths []string) error
	TagObjectForTrash(path string, trashedAt time.Time) error
	RemoveTrashMarker(path string) error
}
