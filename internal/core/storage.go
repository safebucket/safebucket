package core

import (
	"api/internal/models"
	"api/internal/storage"
)

func NewStorage(config models.StorageConfiguration) storage.IStorage {
	switch config.Type {
	case "s3":
		return storage.NewS3Storage(config)
	case "gcp":
		return storage.NewGCPStorage(config)
	default:
		return nil
	}
}
