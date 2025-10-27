package storage

import (
	"context"
	"errors"
	"net/http"
	"time"

	c "api/internal/configuration"

	gcs "cloud.google.com/go/storage"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type GCPStorage struct {
	BucketName string
	storage    *gcs.Client
}

func NewGCPStorage(bucketName string) IStorage {
	client, err := gcs.NewClient(context.Background())
	if err != nil {
		zap.L().Error("Failed to connect to storage", zap.Error(err))
	}

	_, err = client.Bucket(bucketName).Attrs(context.Background())
	if err != nil {
		zap.L().
			Error("Failed to retrieve bucket.", zap.String("bucketName", bucketName), zap.Error(err))
	}

	return &GCPStorage{
		BucketName: bucketName,
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

func (g GCPStorage) PresignedPostPolicy(
	path string,
	size int,
	metadata map[string]string,
) (string, map[string]string, error) {
	opts := &gcs.PostPolicyV4Options{
		Expires: time.Now().Add(c.UploadPolicyExpirationInMinutes * time.Minute),
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

func (g GCPStorage) StatObject(path string) (map[string]string, error) {
	file, err := g.storage.Bucket(g.BucketName).Object(path).Attrs(context.Background())
	if err != nil {
		return nil, err
	}

	return file.Metadata, err
}

func (g GCPStorage) RemoveObject(path string) error {
	return g.storage.Bucket(g.BucketName).Object(path).Delete(context.Background())
}

func (g GCPStorage) RemoveObjects(paths []string) error {
	// GCP doesn't have native batch delete, so we delete one by one
	for _, path := range paths {
		if err := g.RemoveObject(path); err != nil {
			zap.L().Error("Failed to delete object", zap.String("key", path), zap.Error(err))
			return err
		}
	}
	return nil
}

func (g GCPStorage) ListObjects(prefix string, _ int32) ([]string, error) {
	bucket := g.storage.Bucket(g.BucketName)

	query := &gcs.Query{
		Prefix: prefix,
	}

	it := bucket.Objects(context.Background(), query)

	var objects []string

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		objects = append(objects, attrs.Name)
	}

	return objects, nil
}

func (g GCPStorage) SetObjectTags(path string, tags map[string]string) error {
	obj := g.storage.Bucket(g.BucketName).Object(path)

	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		return err
	}

	if attrs.Metadata == nil {
		attrs.Metadata = make(map[string]string)
	}

	for key, value := range tags {
		attrs.Metadata[key] = value
	}

	_, err = obj.Update(context.Background(), gcs.ObjectAttrsToUpdate{
		Metadata: attrs.Metadata,
	})
	return err
}

func (g GCPStorage) GetObjectTags(path string) (map[string]string, error) {
	obj := g.storage.Bucket(g.BucketName).Object(path)

	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		return nil, err
	}

	if attrs.Metadata == nil {
		return make(map[string]string), nil
	}

	tagMap := make(map[string]string)
	for key, value := range attrs.Metadata {
		tagMap[key] = value
	}

	return tagMap, nil
}

func (g GCPStorage) RemoveObjectTags(path string, tagsToRemove []string) error {
	obj := g.storage.Bucket(g.BucketName).Object(path)

	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		return err
	}

	if attrs.Metadata != nil {
		for _, key := range tagsToRemove {
			delete(attrs.Metadata, key)
		}
	}

	_, err = obj.Update(context.Background(), gcs.ObjectAttrsToUpdate{
		Metadata: attrs.Metadata,
	})
	return err
}
