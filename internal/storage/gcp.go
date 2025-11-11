package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	c "api/internal/configuration"
	"api/internal/models"

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

func (g GCPStorage) GetBucketName() string {
	return g.BucketName
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

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Pattern: trash/{bucket-id}/{rest} -> buckets/{bucket-id}/{rest}.
func (g GCPStorage) IsTrashMarkerPath(path string) (bool, string) {
	if strings.HasPrefix(path, trashPrefix) {
		originalPath := bucketsPrefix + strings.TrimPrefix(path, trashPrefix)
		return true, originalPath
	}

	return false, ""
}

// getTrashMarkerPath converts buckets/{id}/path to trash/{id}/path.
func (g GCPStorage) getTrashMarkerPath(objectPath string) string {
	return strings.Replace(objectPath, bucketsPrefix, trashPrefix, 1)
}

func (g GCPStorage) MarkFileAsTrashed(objectPath string, metadata models.TrashMetadata) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath)

	// Only verify object exists for files (not folders, which only exist in database)
	if !metadata.IsFolder {
		obj := g.storage.Bucket(g.BucketName).Object(objectPath)
		if _, err := obj.Attrs(ctx); err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	// Create empty marker object to trigger lifecycle policy deletion
	markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)
	writer := markerObj.NewWriter(ctx)

	// Write empty content (0 bytes)
	if _, err := writer.Write([]byte{}); err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	return nil
}

func (g GCPStorage) UnmarkFileAsTrashed(objectPath string) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath)

	markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)
	if err := markerObj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to remove marker: %w", err)
	}

	return nil
}

func (g GCPStorage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	ctx := context.Background()
	bucket := g.storage.Bucket(g.BucketName)

	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		zap.L().Error("Failed to get bucket attributes",
			zap.String("bucket", g.BucketName),
			zap.Error(err))
		return err
	}

	const trashRuleActionType = gcs.DeleteAction
	const multipartRuleActionType = gcs.AbortIncompleteMPUAction
	var existingTrashRuleIndex = -1
	var existingMultipartRuleIndex = -1

	if attrs.Lifecycle.Rules != nil {
		for i, rule := range attrs.Lifecycle.Rules {
			// Check for trash expiration rule
			if rule.Action.Type == trashRuleActionType &&
				rule.Condition.MatchesPrefix != nil &&
				len(rule.Condition.MatchesPrefix) > 0 &&
				rule.Condition.MatchesPrefix[0] == trashPrefix {
				existingTrashRuleIndex = i

				if rule.Condition.AgeInDays == int64(retentionDays) {
					zap.L().Debug("Trash lifecycle policy already up-to-date",
						zap.String("bucket", g.BucketName),
						zap.Int("retentionDays", retentionDays))
					// Don't return yet - need to check multipart rule too
				}
			}

			// Check for multipart upload cleanup rule
			if rule.Action.Type == multipartRuleActionType &&
				rule.Condition.AgeInDays == 1 {
				existingMultipartRuleIndex = i
				zap.L().Debug("Multipart upload cleanup policy already up-to-date",
					zap.String("bucket", g.BucketName))
			}
		}
	}

	trashRule := gcs.LifecycleRule{
		Action: gcs.LifecycleAction{
			Type: trashRuleActionType,
		},
		Condition: gcs.LifecycleCondition{
			AgeInDays:     int64(retentionDays),
			MatchesPrefix: []string{trashPrefix},
		},
	}

	multipartRule := gcs.LifecycleRule{
		Action: gcs.LifecycleAction{
			Type: multipartRuleActionType,
		},
		Condition: gcs.LifecycleCondition{
			AgeInDays: 1,
		},
	}

	var newRules []gcs.LifecycleRule
	if attrs.Lifecycle.Rules != nil {
		for i, rule := range attrs.Lifecycle.Rules {
			// Skip existing trash and multipart rules - we'll add updated versions
			if i != existingTrashRuleIndex && i != existingMultipartRuleIndex {
				newRules = append(newRules, rule)
			}
		}
	}
	newRules = append(newRules, trashRule, multipartRule)

	updateAttrs := gcs.BucketAttrsToUpdate{
		Lifecycle: &gcs.Lifecycle{
			Rules: newRules,
		},
	}

	if _, err = bucket.Update(ctx, updateAttrs); err != nil {
		zap.L().Error("Failed to update lifecycle policies",
			zap.String("bucket", g.BucketName),
			zap.Int("trashRetentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	zap.L().Info("Lifecycle policies configured",
		zap.String("bucket", g.BucketName),
		zap.Int("trashRetentionDays", retentionDays),
		zap.Int("multipartCleanupDays", 1))

	return nil
}
