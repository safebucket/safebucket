package storage

import (
	"path"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
)

func VersionObjectKey(bucketID, versionID uuid.UUID) string {
	return path.Join("buckets", bucketID.String(), versionID.String())
}

func fileContentPath(objectPath string, object any) string {
	file, ok := object.(models.File)
	if !ok {
		return objectPath
	}
	return VersionObjectKey(file.BucketID, file.ContentVersionID())
}
