package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
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
		zap.L().Fatal("Failed to create storage client", zap.Error(err))
	}

	_, err = client.Bucket(bucketName).Attrs(context.Background())
	if err != nil {
		zap.L().Fatal("Failed to connect to storage or bucket does not exist",
			zap.String("bucketName", bucketName),
			zap.Error(err))
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
		Expires: time.Now().Add(c.UploadPolicyExpirationInMinutes * time.Minute),
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

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Patterns:
//   - trash/{bucket-id}/files/{file-id} -> buckets/{bucket-id}/{file-id}
//   - trash/{bucket-id}/folders/{folder-id} -> buckets/{bucket-id}/{folder-id}
func (g GCPStorage) IsTrashMarkerPath(path string) (bool, string) {
	if !strings.HasPrefix(path, trashPrefix) {
		return false, ""
	}

	// Remove "trash/" prefix
	remainder := strings.TrimPrefix(path, trashPrefix)
	parts := strings.SplitN(remainder, "/", 3)

	if len(parts) < 3 {
		return false, ""
	}

	bucketID := parts[0]
	resourceType := parts[1] // "files" or "folders"
	resourceID := parts[2]

	// Validate resource type
	if resourceType != "files" && resourceType != "folders" {
		return false, ""
	}

	// Reconstruct original path: buckets/{bucket-id}/{resource-id}
	originalPath := bucketsPrefix + bucketID + "/" + resourceID
	return true, originalPath
}

// getTrashMarkerPath converts buckets/{bucket-id}/{id} to trash/{bucket-id}/files|folders/{id}.
func (g GCPStorage) getTrashMarkerPath(objectPath string, model interface{}) string {
	// Remove "buckets/" prefix
	remainder := strings.TrimPrefix(objectPath, bucketsPrefix)

	var resourceType string
	switch model.(type) {
	case models.Folder:
		resourceType = folderPath
	case models.File:
		resourceType = filePath
	default:
		return ""
	}

	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) < 2 {
		return ""
	}

	bucketID := parts[0]
	resourceID := parts[1]

	// Pattern: trash/{bucket-id}/files|folders/{resource-id}
	return path.Join(trashPrefix, bucketID, resourceType, resourceID)
}

func (g GCPStorage) MarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath, object)

	if _, ok := object.(models.File); ok {
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

func (g GCPStorage) UnmarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath, object)

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
