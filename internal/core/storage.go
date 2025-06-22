package core

import (
	"api/internal/models"
	"api/internal/storage"
)

func NewStorage(config models.StorageConfiguration) storage.IStorage {
	switch config.Type {
	case "minio":
		return storage.NewS3Storage(config.Minio, config.Minio.BucketName)
	case "gcp":
		return storage.NewGCPStorage(config.GCP.BucketName)
	case "aws":
		return storage.NewAWSStorage(config.AWS.BucketName)
	default:
		return nil
	}
}
