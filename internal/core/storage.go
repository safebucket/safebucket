package core

import (
	"api/internal/models"
	"api/internal/storage"
)

func NewStorage(config models.StorageConfiguration) storage.IStorage {
	switch config.Type {
	case "minio":
		return storage.NewS3Storage(config.Minio, config.BucketName)
	case "gcp":
		return storage.NewGCPStorage(config.BucketName)
	default:
		return nil
	}
}
