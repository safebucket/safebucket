package storage

import (
	"api/internal/models"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
	"time"
)

type S3Storage struct {
	BucketName string
	storage    *minio.Client
}

func NewS3Storage(config models.StorageConfiguration) IStorage {
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

	return S3Storage{BucketName: "safebucket", storage: minioClient}
}

func (s S3Storage) PresignedGetObject(path string) (string, error) {
	url, err := s.storage.PresignedGetObject(context.Background(), s.BucketName, path, time.Minute*15, nil)

	if err != nil {
		return "", err
	}

	return url.String(), nil
}

func (s S3Storage) PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error) {
	policy := minio.NewPostPolicy()
	_ = policy.SetBucket("safebucket") //TODO: set var
	_ = policy.SetKey(path)
	_ = policy.SetContentLengthRange(int64(size), int64(size))
	_ = policy.SetExpires(time.Now().UTC().Add(15 * time.Minute))
	_ = policy.SetUserMetadata("Bucket-Id", metadata["bucket_id"])
	_ = policy.SetUserMetadata("File-Id", metadata["file_id"])
	_ = policy.SetUserMetadata("User-Id", metadata["user_id"])

	url, metadata, err := s.storage.PresignedPostPolicy(context.Background(), policy)

	if err != nil {
		return "", map[string]string{}, err
	}

	return url.String(), metadata, nil
}

func (s S3Storage) StatObject(path string) error {
	_, err := s.storage.StatObject(context.Background(), s.BucketName, path, minio.StatObjectOptions{})
	return err
}

func (s S3Storage) RemoveObject(path string) error {
	return s.storage.RemoveObject(context.Background(), s.BucketName, path, minio.RemoveObjectOptions{})
}
