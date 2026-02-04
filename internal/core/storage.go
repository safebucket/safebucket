package core

import (
	"api/internal/models"
	"api/internal/storage"

	"go.uber.org/zap"
)

func NewStorage(config models.StorageConfiguration, trashRetentionDays int) storage.IStorage {
	var store storage.IStorage

	switch config.Type {
	case ProviderMinio:
		store = storage.NewS3Storage(config.Minio, config.Minio.BucketName)
	case ProviderGCP:
		store = storage.NewGCPStorage(config.CloudStorage.BucketName)
	case ProviderAWS:
		store = storage.NewAWSStorage(config.AWS.BucketName)
	case ProviderRustFS:
		store = storage.NewRustFSStorage(config.RustFS, config.RustFS.BucketName)
	case ProviderS3:
		store = storage.NewGenericS3Storage(config.S3, config.S3.BucketName)
	default:
		return nil
	}

	if store != nil && trashRetentionDays > 0 {
		err := store.EnsureTrashLifecyclePolicy(trashRetentionDays)
		if err != nil {
			zap.L().Fatal("Failed to configure trash lifecycle policy",
				zap.String("provider", config.Type),
				zap.Int("retentionDays", trashRetentionDays),
				zap.Error(err))
		}
	}

	return store
}
