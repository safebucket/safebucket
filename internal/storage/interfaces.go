package storage

import "api/internal/models"

const (
	bucketsPrefix = "buckets/"
	trashPrefix   = "trash/"
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
	SetObjectTags(path string, tags map[string]string) error
	GetObjectTags(path string) (map[string]string, error)
	RemoveObjectTags(path string, tagsToRemove []string) error
	EnsureTrashLifecyclePolicy(retentionDays int) error
	MarkFileAsTrashed(objectPath string, metadata models.TrashMetadata) error
	UnmarkFileAsTrashed(objectPath string) error
	IsTrashMarkerPath(path string) (isMarker bool, originalPath string)
	GetBucketName() string
}
