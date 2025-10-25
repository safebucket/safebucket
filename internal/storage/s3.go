package storage

import (
	c "api/internal/configuration"
	"api/internal/models"
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/tags"
	"go.uber.org/zap"
)

type S3Storage struct {
	BucketName string
	storage    *minio.Client
}

func NewS3Storage(config *models.MinioStorageConfiguration, bucketName string) IStorage {
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.ClientId, config.ClientSecret, ""),
		Secure: false,
	})

	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	if !exists {
		zap.L().Error("Failed to retrieve bucket.", zap.String("bucketName", bucketName), zap.Error(err))
	}

	return S3Storage{BucketName: bucketName, storage: minioClient}
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
	_ = policy.SetBucket(s.BucketName)
	_ = policy.SetKey(path)
	_ = policy.SetContentLengthRange(int64(size), int64(size))
	_ = policy.SetExpires(time.Now().UTC().Add(c.UploadPolicyExpirationInMinutes * time.Minute))
	_ = policy.SetUserMetadata("Bucket-Id", metadata["bucket_id"])
	_ = policy.SetUserMetadata("File-Id", metadata["file_id"])
	_ = policy.SetUserMetadata("User-Id", metadata["user_id"])

	url, metadata, err := s.storage.PresignedPostPolicy(context.Background(), policy)

	if err != nil {
		return "", map[string]string{}, err
	}

	return url.String(), metadata, nil
}

func (s S3Storage) StatObject(path string) (map[string]string, error) {
	file, err := s.storage.StatObject(context.Background(), s.BucketName, path, minio.StatObjectOptions{})

	if err != nil {
		return nil, err
	}

	return file.UserMetadata, err
}

func (s S3Storage) ListObjects(prefix string, maxKeys int32) ([]string, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
		MaxKeys:   int(maxKeys),
	}

	var objects []string

	for object := range s.storage.ListObjects(context.Background(), s.BucketName, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

func (s S3Storage) RemoveObject(path string) error {
	return s.storage.RemoveObject(context.Background(), s.BucketName, path, minio.RemoveObjectOptions{})
}

func (s S3Storage) RemoveObjects(paths []string) error {
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, path := range paths {
			objectsCh <- minio.ObjectInfo{Key: path}
		}
	}()

	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}

	errorCh := s.storage.RemoveObjects(context.Background(), s.BucketName, objectsCh, opts)

	for err := range errorCh {
		if err.Err != nil {
			zap.L().Error("Failed to delete object", zap.String("key", err.ObjectName), zap.Error(err.Err))
			return err.Err
		}
	}

	return nil
}

func (s S3Storage) SetObjectTags(path string, tagMap map[string]string) error {
	objectTags, err := tags.MapToObjectTags(tagMap)
	if err != nil {
		return err
	}

	err = s.storage.PutObjectTagging(context.Background(), s.BucketName, path, objectTags, minio.PutObjectTaggingOptions{})
	return err
}

func (s S3Storage) GetObjectTags(path string) (map[string]string, error) {
	currentTags, err := s.storage.GetObjectTagging(context.Background(), s.BucketName, path, minio.GetObjectTaggingOptions{})
	if err != nil {
		return nil, err
	}

	return currentTags.ToMap(), nil
}

func (s S3Storage) RemoveObjectTags(path string, tagsToRemove []string) error {
	currentTags, err := s.storage.GetObjectTagging(context.Background(), s.BucketName, path, minio.GetObjectTaggingOptions{})
	if err != nil {
		return err
	}

	tagMap := currentTags.ToMap()

	for _, tagKey := range tagsToRemove {
		delete(tagMap, tagKey)
	}

	filteredTags, err := tags.MapToObjectTags(tagMap)
	if err != nil {
		return err
	}

	err = s.storage.PutObjectTagging(context.Background(), s.BucketName, path, filteredTags, minio.PutObjectTaggingOptions{})
	return err
}
