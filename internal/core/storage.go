package core

import (
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"go.uber.org/zap"
)

func NewStorage(config models.StorageConfiguration) storage.IStorage {
	var store storage.IStorage

	switch config.Type {
	case configuration.ProviderMinio:
		store = storage.NewS3Storage(config.Minio)
	case configuration.ProviderGCP:
		store = storage.NewGCPStorage(config.CloudStorage.BucketName)
	case configuration.ProviderAWS:
		store = storage.NewAWSStorage(config.AWS.BucketName)
	case configuration.ProviderRustFS:
		store = storage.NewRustFSStorage(config.RustFS)
	case configuration.ProviderS3:
		store = storage.NewGenericS3Storage(config.S3)
	case configuration.ProviderAzure:
		store = storage.NewAzureStorage(config.Azure)
	default:
		return nil
	}

	if err := store.EnsureMultipartCleanupPolicy(); err != nil {
		zap.L().Fatal("Failed to configure multipart cleanup policy",
			zap.String("provider", config.Type),
			zap.Error(err))
	}

	return store
}
