package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	c "api/internal/configuration"
	"api/internal/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/tags"
	"go.uber.org/zap"
)

type S3Storage struct {
	BucketName       string
	InternalEndpoint string
	ExternalEndpoint string
	storage          *minio.Client
}

// s3Config holds a common configuration for S3-compatible storage providers.
type s3Config struct {
	endpoint         string
	externalEndpoint string
	accessKey        string
	secretKey        string
	providerName     string
}

// newS3Storage creates a new S3-compatible storage client.
func newS3Storage(cfg s3Config, bucketName string) IStorage {
	minioClient, err := minio.New(cfg.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.accessKey, cfg.secretKey, ""),
		Secure: false,
	})
	if err != nil {
		zap.L().Error("Failed to connect to storage",
			zap.String("provider", cfg.providerName),
			zap.Error(err))
	}

	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		zap.L().Error("Failed to connect to storage",
			zap.String("provider", cfg.providerName),
			zap.Error(err))
	}

	if !exists {
		zap.L().Error("Failed to retrieve bucket",
			zap.String("provider", cfg.providerName),
			zap.String("bucketName", bucketName),
			zap.Error(err))
	}

	externalEndpoint := cfg.externalEndpoint
	if externalEndpoint == "" {
		externalEndpoint = cfg.endpoint
	}

	return S3Storage{
		BucketName:       bucketName,
		InternalEndpoint: cfg.endpoint,
		ExternalEndpoint: externalEndpoint,
		storage:          minioClient,
	}
}

func NewS3Storage(config *models.MinioStorageConfiguration, bucketName string) IStorage {
	return newS3Storage(s3Config{
		endpoint:         config.Endpoint,
		externalEndpoint: config.ExternalEndpoint,
		accessKey:        config.ClientID,
		secretKey:        config.ClientSecret,
		providerName:     "MinIO",
	}, bucketName)
}

func NewRustFSStorage(config *models.RustFSStorageConfiguration, bucketName string) IStorage {
	return newS3Storage(s3Config{
		endpoint:         config.Endpoint,
		externalEndpoint: config.ExternalEndpoint,
		accessKey:        config.AccessKey,
		secretKey:        config.SecretKey,
		providerName:     "RustFS",
	}, bucketName)
}

// replaceEndpoint replaces the internal endpoint with the external endpoint in a URL.
// It properly parses URLs to replace only the scheme and host, preserving path and query parameters.
func (s S3Storage) replaceEndpoint(urlString string) string {
	if s.InternalEndpoint == s.ExternalEndpoint {
		return urlString
	}

	presignedURL, err := url.Parse(urlString)
	if err != nil {
		zap.L().Warn("failed to parse presigned URL, using original", zap.Error(err))
		return urlString
	}

	externalURL, err := url.Parse(s.ExternalEndpoint)
	if err != nil {
		zap.L().Warn("failed to parse external endpoint, using original URL", zap.Error(err))
		return urlString
	}

	presignedURL.Scheme = externalURL.Scheme
	presignedURL.Host = externalURL.Host

	return presignedURL.String()
}

func (s S3Storage) GetBucketName() string {
	return s.BucketName
}

func (s S3Storage) PresignedGetObject(path string) (string, error) {
	presignedURL, err := s.storage.PresignedGetObject(
		context.Background(),
		s.BucketName,
		path,
		time.Minute*15,
		nil,
	)
	if err != nil {
		return "", err
	}

	// Replace internal endpoint with external endpoint for browser access
	urlString := s.replaceEndpoint(presignedURL.String())
	return urlString, nil
}

func (s S3Storage) PresignedPostPolicy(
	path string,
	size int,
	metadata map[string]string,
) (string, map[string]string, error) {
	policy := minio.NewPostPolicy()
	_ = policy.SetBucket(s.BucketName)
	_ = policy.SetKey(path)
	_ = policy.SetContentLengthRange(int64(size), int64(size))
	_ = policy.SetExpires(time.Now().UTC().Add(c.UploadPolicyExpirationInMinutes * time.Minute))
	_ = policy.SetUserMetadata("Bucket-Id", metadata["bucket_id"])
	_ = policy.SetUserMetadata("File-Id", metadata["file_id"])
	_ = policy.SetUserMetadata("User-Id", metadata["user_id"])

	presignedURL, metadata, err := s.storage.PresignedPostPolicy(context.Background(), policy)
	if err != nil {
		return "", map[string]string{}, err
	}

	// Replace internal endpoint with external endpoint for browser access
	urlString := s.replaceEndpoint(presignedURL.String())
	return urlString, metadata, nil
}

func (s S3Storage) StatObject(path string) (map[string]string, error) {
	file, err := s.storage.StatObject(
		context.Background(),
		s.BucketName,
		path,
		minio.StatObjectOptions{},
	)
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
	return s.storage.RemoveObject(
		context.Background(),
		s.BucketName,
		path,
		minio.RemoveObjectOptions{},
	)
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
			zap.L().
				Error("Failed to delete object", zap.String("key", err.ObjectName), zap.Error(err.Err))
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

	err = s.storage.PutObjectTagging(
		context.Background(),
		s.BucketName,
		path,
		objectTags,
		minio.PutObjectTaggingOptions{},
	)
	return err
}

func (s S3Storage) GetObjectTags(path string) (map[string]string, error) {
	currentTags, err := s.storage.GetObjectTagging(
		context.Background(),
		s.BucketName,
		path,
		minio.GetObjectTaggingOptions{},
	)
	if err != nil {
		return nil, err
	}

	return currentTags.ToMap(), nil
}

func (s S3Storage) RemoveObjectTags(path string, tagsToRemove []string) error {
	currentTags, err := s.storage.GetObjectTagging(
		context.Background(),
		s.BucketName,
		path,
		minio.GetObjectTaggingOptions{},
	)
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

	err = s.storage.PutObjectTagging(
		context.Background(),
		s.BucketName,
		path,
		filteredTags,
		minio.PutObjectTaggingOptions{},
	)
	return err
}

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Patterns:
//   - trash/{bucket-id}/files/{file-id} -> buckets/{bucket-id}/{file-id}
//   - trash/{bucket-id}/folders/{folder-id} -> buckets/{bucket-id}/{folder-id}
func (s S3Storage) IsTrashMarkerPath(path string) (bool, string) {
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
	if resourceType != folderPath && resourceType != filePath {
		return false, ""
	}

	// Reconstruct original path: buckets/{bucket-id}/{resource-id}
	originalPath := bucketsPrefix + bucketID + "/" + resourceID
	return true, originalPath
}

// getTrashMarkerPath converts buckets/{bucket-id}/{id} to trash/{bucket-id}/files|folders/{id}.
func (s S3Storage) getTrashMarkerPath(objectPath string, model interface{}) string {
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

func (s S3Storage) MarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := s.getTrashMarkerPath(objectPath, object)

	if _, ok := object.(models.File); ok {
		_, err := s.storage.StatObject(ctx, s.BucketName, objectPath, minio.StatObjectOptions{})
		if err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	// Create empty marker object to trigger lifecycle policy deletion
	reader := bytes.NewReader([]byte{})
	_, err := s.storage.PutObject(ctx, s.BucketName, markerPath, reader, 0, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}
	return nil
}

func (s S3Storage) UnmarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := s.getTrashMarkerPath(objectPath, object)
	err := s.storage.RemoveObject(ctx, s.BucketName, markerPath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove marker: %w", err)
	}
	return nil
}

// processExistingLifecycleRules processes existing lifecycle rules and returns the updated configuration.
func (s S3Storage) processExistingLifecycleRules(
	existingConfig *lifecycle.Configuration,
	err error,
	trashRuleID, multipartRuleID string,
	retentionDays int,
) *lifecycle.Configuration {
	if err != nil || existingConfig == nil || existingConfig.Empty() {
		return lifecycle.NewConfiguration()
	}

	config := existingConfig
	var newRules []lifecycle.Rule
	trashRuleFound := false
	multipartRuleFound := false

	for _, rule := range config.Rules {
		switch rule.ID {
		case trashRuleID:
			trashRuleFound = true
			if !rule.Expiration.IsDaysNull() &&
				int(rule.Expiration.Days) == retentionDays &&
				rule.RuleFilter.Prefix == "trash/" {
				zap.L().Debug("Trash lifecycle policy already up-to-date",
					zap.String("bucket", s.BucketName),
					zap.Int("retentionDays", retentionDays))
				newRules = append(newRules, rule)
			}

		case multipartRuleID:
			multipartRuleFound = true
			if rule.AbortIncompleteMultipartUpload.DaysAfterInitiation == 1 {
				zap.L().Debug("Multipart upload cleanup policy already up-to-date",
					zap.String("bucket", s.BucketName))
				newRules = append(newRules, rule)
			}

		default:
			newRules = append(newRules, rule)
		}
	}

	if trashRuleFound || multipartRuleFound {
		config.Rules = newRules
	}

	return config
}

// EnsureTrashLifecyclePolicy configures lifecycle policies for the bucket, merging with existing rules.
// It adds or updates the trash expiration rule (prefix: trash/) with the specified retention period.
//
// NOTE: AbortIncompleteMultipartUpload is not supported by MinIO.
// MinIO does not fully support the AbortIncompleteMultipartUpload lifecycle action.
// References:
// - https://github.com/minio/minio/issues/16120
// - https://github.com/minio/minio/issues/19115
func (s S3Storage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	const trashRuleID = "safebucket-trash-retention"
	const multipartRuleID = "safebucket-abort-incomplete-multipart"

	// Validate retentionDays to prevent overflow and invalid values
	if retentionDays < 0 {
		return fmt.Errorf("retentionDays %d cannot be negative", retentionDays)
	}
	if retentionDays > 2147483647 { // math.MaxInt32
		return fmt.Errorf("retentionDays %d exceeds maximum allowed value (2147483647)", retentionDays)
	}

	ctx := context.Background()

	// Fetch existing lifecycle configuration
	existingConfig, err := s.storage.GetBucketLifecycle(ctx, s.BucketName)

	// Process existing rules to preserve non-SafeBucket policies
	config := s.processExistingLifecycleRules(
		existingConfig,
		err,
		trashRuleID,
		multipartRuleID,
		retentionDays,
	)

	// Check if trash rule already exists and is up-to-date
	trashRuleExists := false
	for _, rule := range config.Rules {
		if rule.ID == trashRuleID {
			trashRuleExists = true
			break
		}
	}

	// Add or update trash rule if needed
	if !trashRuleExists {
		trashRule := lifecycle.Rule{
			ID:     trashRuleID,
			Status: "Enabled",
			RuleFilter: lifecycle.Filter{
				Prefix: "trash/",
			},
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(retentionDays),
			},
		}
		config.Rules = append(config.Rules, trashRule)
	}

	err = s.storage.SetBucketLifecycle(ctx, s.BucketName, config)
	if err != nil {
		zap.L().Error("Failed to set lifecycle policies",
			zap.String("bucket", s.BucketName),
			zap.Int("trashRetentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	zap.L().Info("Lifecycle policies configured",
		zap.String("bucket", s.BucketName),
		zap.Int("trashRetentionDays", retentionDays),
		zap.Int("multipartCleanupDays", 1))
	return nil
}
