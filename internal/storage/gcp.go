package storage

import (
	"api/internal/models"
	gcs "cloud.google.com/go/storage"
	"context"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type GCPStorage struct {
	BucketName string
	storage    *gcs.Client
}

func NewGCPStorage(config models.StorageConfiguration) IStorage {
	client, err := gcs.NewClient(context.Background())
	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	_, err = client.Bucket(config.BucketName).Attrs(context.Background())
	if err != nil {
		zap.L().Error("Bucket does not exist.", zap.String("bucketName", config.BucketName))
	}

	return &GCPStorage{
		BucketName: config.BucketName,
		storage:    client,
	}
}

func (g GCPStorage) PresignedGetObject(path string) (string, error) {
	opts := &gcs.SignedURLOptions{
		Method:  http.MethodGet,
		Expires: time.Now().Add(15 * time.Minute),
	}

	url, err := g.storage.Bucket(g.BucketName).SignedURL(path, opts)

	if err != nil {
		return "", err
	}

	return url, nil
}

func (g GCPStorage) PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error) {
	opts := &gcs.PostPolicyV4Options{
		Expires: time.Now().Add(15 * time.Minute),
		Fields: &gcs.PolicyV4Fields{
			Metadata: map[string]string{
				"x-goog-meta-bucket-id": metadata["bucket_id"],
				"x-goog-meta-file-id":   metadata["file_id"],
				"x-goog-meta-user-id":   metadata["user_id"],
			},
		},
		Conditions: []gcs.PostPolicyV4Condition{
			gcs.ConditionContentLengthRange(uint64(size), uint64(size)), // #nosec G115
		},
	}

	postPolicy, err := g.storage.Bucket(g.BucketName).GenerateSignedPostPolicyV4(path, opts)

	if err != nil {
		zap.L().Error("Failed to generate post policy", zap.Error(err))
		return "", nil, err
	}

	return postPolicy.URL, postPolicy.Fields, nil
}

func (g GCPStorage) StatObject(path string) error {
	_, err := g.storage.Bucket(g.BucketName).Object(path).Attrs(context.Background())
	return err
}

func (g GCPStorage) RemoveObject(path string) error {
	return g.storage.Bucket(g.BucketName).Object(path).Delete(context.Background())
}
