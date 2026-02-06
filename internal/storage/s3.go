package storage

import (
	"context"
	"net/url"
	"time"

	c "api/internal/configuration"
	"api/internal/models"

	"strings"

	"path"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// GenericS3Storage implements IStorage for generic S3-compatible providers
// (Storj, Hetzner, Backblaze B2, Garage, etc.).
// This provider assumes NO lifecycle policy or bucket notification support.
type GenericS3Storage struct {
	BucketName       string
	InternalEndpoint string
	ExternalEndpoint string
	Region           string
	storage          *minio.Client
}

// NewGenericS3Storage creates a new generic S3-compatible storage client.
func NewGenericS3Storage(config *models.S3Configuration, bucketName string) IStorage {
	endpoint := config.Endpoint

	minioOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseTLS,
		Region: config.Region,
	}

	if config.ForcePathStyle {
		minioOptions.BucketLookup = minio.BucketLookupPath
	}

	minioClient, err := minio.New(endpoint, minioOptions)
	if err != nil {
		zap.L().Fatal("Failed to create S3 storage client", zap.Error(err))
	}

	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		zap.L().Fatal("Failed to connect to S3 storage", zap.Error(err))
	}

	if !exists {
		zap.L().Fatal("S3 bucket does not exist", zap.String("bucketName", bucketName))
	}

	return &GenericS3Storage{
		BucketName:       bucketName,
		InternalEndpoint: endpoint,
		ExternalEndpoint: config.ExternalEndpoint,
		Region:           config.Region,
		storage:          minioClient,
	}
}

// replaceEndpoint replaces the internal endpoint with the external endpoint in a URL.
func (s *GenericS3Storage) replaceEndpoint(urlString string) string {
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

func (s *GenericS3Storage) GetBucketName() string {
	return s.BucketName
}

func (s *GenericS3Storage) PresignedGetObject(objectPath string) (string, error) {
	presignedURL, err := s.storage.PresignedGetObject(
		context.Background(),
		s.BucketName,
		objectPath,
		c.UploadPolicyExpirationInMinutes*time.Minute,
		nil,
	)
	if err != nil {
		return "", err
	}

	urlString := s.replaceEndpoint(presignedURL.String())
	return urlString, nil
}

func (s *GenericS3Storage) PresignedPostPolicy(
	objectPath string,
	size int,
	metadata map[string]string,
) (string, map[string]string, error) {
	policy := minio.NewPostPolicy()
	_ = policy.SetBucket(s.BucketName)
	_ = policy.SetKey(objectPath)
	_ = policy.SetContentLengthRange(int64(size), int64(size))
	_ = policy.SetExpires(time.Now().UTC().Add(c.UploadPolicyExpirationInMinutes * time.Minute))
	_ = policy.SetUserMetadata("Bucket-Id", metadata["bucket_id"])
	_ = policy.SetUserMetadata("File-Id", metadata["file_id"])
	_ = policy.SetUserMetadata("User-Id", metadata["user_id"])

	presignedURL, formData, err := s.storage.PresignedPostPolicy(context.Background(), policy)
	if err != nil {
		return "", map[string]string{}, err
	}

	urlString := s.replaceEndpoint(presignedURL.String())
	return urlString, formData, nil
}

func (s *GenericS3Storage) StatObject(objectPath string) (map[string]string, error) {
	file, err := s.storage.StatObject(
		context.Background(),
		s.BucketName,
		objectPath,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	return file.UserMetadata, err
}

func (s *GenericS3Storage) ListObjects(prefix string, maxKeys int32) ([]string, error) {
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

func (s *GenericS3Storage) RemoveObject(objectPath string) error {
	return s.storage.RemoveObject(
		context.Background(),
		s.BucketName,
		objectPath,
		minio.RemoveObjectOptions{},
	)
}

func (s *GenericS3Storage) RemoveObjects(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// Some S3 API enforces max 1000 objects per DeleteObjects call
	for i := 0; i < len(paths); i += c.BulkActionsLimit {
		end := i + c.BulkActionsLimit
		if end > len(paths) {
			end = len(paths)
		}

		batch := paths[i:end]
		objectsCh := make(chan minio.ObjectInfo, len(batch))

		for _, p := range batch {
			objectsCh <- minio.ObjectInfo{Key: p}
		}
		close(objectsCh)

		opts := minio.RemoveObjectsOptions{GovernanceBypass: true}
		errorCh := s.storage.RemoveObjects(context.Background(), s.BucketName, objectsCh, opts)

		for err := range errorCh {
			if err.Err != nil {
				zap.L().Error("Failed to delete object",
					zap.String("key", err.ObjectName),
					zap.Int("batchStart", i),
					zap.Error(err.Err))
				return err.Err
			}
		}
	}

	return nil
}

// IsTrashMarkerPath checks if the given path is a trash marker and returns the original object path.
// Generic S3 providers lack lifecycle policies, so the trash worker triggers the expiration manually.
func (s *GenericS3Storage) IsTrashMarkerPath(markerPath string) (bool, string) {
	if !strings.HasPrefix(markerPath, trashPrefix) {
		return false, ""
	}

	// Remove "trash/" prefix
	remainder := strings.TrimPrefix(markerPath, trashPrefix)
	parts := strings.SplitN(remainder, "/", 3)

	if len(parts) < 3 {
		return false, ""
	}

	bucketID := parts[0]
	fileID := parts[2]

	originalPath := path.Join("buckets", bucketID, fileID)
	return true, originalPath
}

// MarkAsTrashed is a no-op for generic S3 providers.
// Trash state is tracked only in the database, not in storage.
func (s *GenericS3Storage) MarkAsTrashed(_ string, _ any) error {
	return nil
}

// UnmarkAsTrashed is a no-op for generic S3 providers.
// Trash state is tracked only in the database, not in storage.
func (s *GenericS3Storage) UnmarkAsTrashed(_ string, _ any) error {
	return nil
}

// EnsureTrashLifecyclePolicy is a no-op for generic S3 providers.
// Most S3-compatible providers (Storj, Hetzner, Backblaze B2, Garage) do not support lifecycle policies.
func (s *GenericS3Storage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	zap.L().Warn("S3 provider does not support lifecycle policies - trash cleanup must be handled manually",
		zap.String("bucket", s.BucketName),
		zap.Int("configuredRetentionDays", retentionDays))
	return nil
}
