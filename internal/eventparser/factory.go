package eventparser

import (
	"strings"

	"api/internal/configuration"
	"api/internal/storage"
)

// NewBucketEventParser returns the appropriate parser for the given storage provider type.
func NewBucketEventParser(storageType string, store storage.IStorage) IBucketEventParser {
	switch storageType {
	case configuration.ProviderRustFS:
		return &RustFSEventParser{}
	case configuration.ProviderMinio:
		return &MinIOEventParser{}
	case configuration.ProviderAWS:
		return &AWSEventParser{Storage: store}
	case configuration.ProviderGCP:
		return &GCPEventParser{}
	case configuration.ProviderS3:
		return &MinIOEventParser{}
	default:
		return &RustFSEventParser{}
	}
}

func ExtractBucketID(objectKey string) string {
	if strings.HasPrefix(objectKey, "buckets/") || strings.HasPrefix(objectKey, "trash/") {
		parts := strings.Split(objectKey, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}
