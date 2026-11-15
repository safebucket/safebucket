package storage

import (
	"path"

	"github.com/google/uuid"
)

func VersionObjectKey(bucketID, versionID uuid.UUID) string {
	return path.Join("buckets", bucketID.String(), versionID.String())
}
