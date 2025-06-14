package core

import (
	"api/internal/models"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func InitStorage(config models.StorageConfiguration) *minio.Client {
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.ClientId, config.ClientSecret, ""),
		Secure: false,
	})

	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	exists, err := minioClient.BucketExists(context.Background(), "safebucket")
	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	if !exists {
		zap.L().Error("Bucket 'safebucket' does not exist.")
	}

	return minioClient
}
