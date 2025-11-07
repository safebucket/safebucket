package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	// GCP lifecycle policies don't support metadata-based conditions
	// Instead, we create a marker file with .trashed extension
	// The lifecycle policy will delete the marker based on age and extension
	if status, exists := tags["Status"]; exists && status == "trashed" {
		// Verify actual file exists before creating marker to prevent orphaned markers
		obj := g.storage.Bucket(g.BucketName).Object(path)
		if _, err := obj.Attrs(context.Background()); err != nil {
			zap.L().Warn("Cannot create .trashed marker for non-existent file",
				zap.String("path", path),
				zap.Error(err))
			return err
		}

		// Create marker file with .trashed extension
		markerPath := path + ".trashed"
		markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)

		writer := markerObj.NewWriter(context.Background())
		// Empty marker file - just acts as a "tag"
		if err := writer.Close(); err != nil {
			return err
		}

		zap.L().Info("Created .trashed marker file",
			zap.String("marker_path", markerPath),
			zap.String("original_path", path))

		return nil
	}

	// For non-trash tags, use metadata (though this won't work with lifecycle policies)
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
	var markerErr error

	// Check if we're removing the "Status" tag (used for trash restoration)
	for _, tag := range tagsToRemove {
		if tag == "Status" {
			// Delete the .trashed marker file
			markerPath := path + ".trashed"
			markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)

			if err := markerObj.Delete(context.Background()); err != nil {
				zap.L().Warn("Failed to delete .trashed marker",
					zap.String("marker_path", markerPath),
					zap.Error(err))
				markerErr = err
			} else {
				zap.L().Info("Deleted .trashed marker file",
					zap.String("marker_path", markerPath),
					zap.String("original_path", path))
			}

			// Continue to remove from metadata as well for consistency
			break
		}
	}

	obj := g.storage.Bucket(g.BucketName).Object(path)

	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		// If marker deletion also failed, return aggregated error
		if markerErr != nil {
			return fmt.Errorf("marker deletion failed: %w; attrs retrieval failed: %v", markerErr, err)
		}
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

	// Aggregate errors if both operations failed
	if err != nil && markerErr != nil {
		return fmt.Errorf("marker deletion failed: %w; metadata update failed: %v", markerErr, err)
	}

	// Return marker error if metadata update succeeded but marker deletion failed
	if err == nil && markerErr != nil {
		return markerErr
	}

	return err
}

// MarkFileAsTrashed creates a .trashed marker file for GCP
// GCP implementation: Creates {path}.trashed marker file
func (g GCPStorage) MarkFileAsTrashed(objectPath string, metadata models.TrashMetadata) error {
	// GCP uses suffix-based markers, reuse existing SetObjectTags logic
	return g.SetObjectTags(objectPath, map[string]string{
		"Status":    "trashed",
		"TrashedAt": metadata.TrashedAt.Format(time.RFC3339),
	})
}

// UnmarkFileAsTrashed removes the .trashed marker file for GCP
// GCP implementation: Deletes the {path}.trashed marker
func (g GCPStorage) UnmarkFileAsTrashed(objectPath string) error {
	// GCP uses suffix-based markers, reuse existing RemoveObjectTags logic
	return g.RemoveObjectTags(objectPath, []string{"Status", "TrashedAt"})
}

// IsTrashMarkerPath checks if a deletion event is for a .trashed marker (GCP uses suffix-based)
// Pattern: {path}.trashed -> original: {path}
func (g GCPStorage) IsTrashMarkerPath(path string) (bool, string) {
	if len(path) > 8 && path[len(path)-8:] == ".trashed" {
		originalPath := path[:len(path)-8]
		return true, originalPath
	}
	return false, ""
}

func (g GCPStorage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	ctx := context.Background()
	bucket := g.storage.Bucket(g.BucketName)

	// Get existing lifecycle configuration
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		zap.L().Error("Failed to get bucket attributes",
			zap.String("bucket", g.BucketName),
			zap.Error(err))
		return err
	}

	// Check if trash lifecycle rule already exists and is up-to-date
	const trashRuleActionType = gcs.DeleteAction
	var existingRuleIndex = -1
	var needsUpdate = true

	if attrs.Lifecycle.Rules != nil {
		for i, rule := range attrs.Lifecycle.Rules {
			// Check if this is our trash rule (matching .trashed suffix)
			if rule.Action.Type == trashRuleActionType &&
				rule.Condition.MatchesSuffix != nil &&
				len(rule.Condition.MatchesSuffix) > 0 &&
				rule.Condition.MatchesSuffix[0] == ".trashed" {

				existingRuleIndex = i

				// Check if retention period matches
				if rule.Condition.AgeInDays == int64(retentionDays) {
					needsUpdate = false
					zap.L().Debug("Trash lifecycle policy already up-to-date",
						zap.String("bucket", g.BucketName),
						zap.Int("retentionDays", retentionDays))
					return nil
				}
				break
			}
		}
	}

	if !needsUpdate {
		return nil
	}

	// Create or update the trash lifecycle rule
	trashRule := gcs.LifecycleRule{
		Action: gcs.LifecycleAction{
			Type: trashRuleActionType,
		},
		Condition: gcs.LifecycleCondition{
			AgeInDays:     int64(retentionDays),
			MatchesSuffix: []string{".trashed"},
		},
	}

	var newRules []gcs.LifecycleRule
	if attrs.Lifecycle.Rules != nil {
		// Preserve existing rules
		for i, rule := range attrs.Lifecycle.Rules {
			if i != existingRuleIndex {
				newRules = append(newRules, rule)
			}
		}
	}
	// Add the new/updated trash rule
	newRules = append(newRules, trashRule)

	// Update bucket lifecycle configuration
	updateAttrs := gcs.BucketAttrsToUpdate{
		Lifecycle: &gcs.Lifecycle{
			Rules: newRules,
		},
	}

	if _, err = bucket.Update(ctx, updateAttrs); err != nil {
		zap.L().Error("Failed to update trash lifecycle policy",
			zap.String("bucket", g.BucketName),
			zap.Int("retentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	zap.L().Info("Trash lifecycle policy configured for GCP",
		zap.String("bucket", g.BucketName),
		zap.Int("retentionDays", retentionDays),
		zap.String("rule", "Delete objects with .trashed suffix"))

	return nil
}
