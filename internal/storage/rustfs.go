package storage

import (
	"context"
	"net/url"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type RustFSStorage struct {
	BucketName    string
	storage       *minio.Client
	signingClient *minio.Client
}

type s3Config struct {
	bucketName       string
	endpoint         string
	externalEndpoint string
	accessKey        string
	secretKey        string
	region           string
	providerName     string
}

func newS3Storage(cfg s3Config) IStorage {
	bucketName := cfg.bucketName
	minioClient, err := minio.New(cfg.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.accessKey, cfg.secretKey, ""),
		Secure: false,
	})
	if err != nil {
		zap.L().Fatal("Failed to create storage client",
			zap.String("provider", cfg.providerName),
			zap.Error(err))
	}

	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		zap.L().Fatal("Failed to connect to storage",
			zap.String("provider", cfg.providerName),
			zap.Error(err))
	}

	if !exists {
		zap.L().Fatal("Bucket does not exist",
			zap.String("provider", cfg.providerName),
			zap.String("bucketName", bucketName))
	}

	externalEndpoint := cfg.externalEndpoint
	if externalEndpoint == "" {
		externalEndpoint = cfg.endpoint
	}

	signingClient, err := newSigningClient(signingClientOptions{
		externalEndpoint: externalEndpoint,
		accessKey:        cfg.accessKey,
		secretKey:        cfg.secretKey,
		region:           cfg.region,
	})
	if err != nil {
		zap.L().Fatal("Failed to create signing client",
			zap.String("provider", cfg.providerName),
			zap.Error(err))
	}

	return RustFSStorage{
		BucketName:    bucketName,
		storage:       minioClient,
		signingClient: signingClient,
	}
}

func NewS3Storage(config *models.MinioStorageConfiguration) IStorage {
	return newS3Storage(s3Config{
		bucketName:       config.BucketName,
		endpoint:         config.Endpoint,
		externalEndpoint: config.ExternalEndpoint,
		accessKey:        config.ClientID,
		secretKey:        config.ClientSecret,
		region:           config.Region,
		providerName:     "MinIO",
	})
}

func NewRustFSStorage(config *models.RustFSStorageConfiguration) IStorage {
	return newS3Storage(s3Config{
		bucketName:       config.BucketName,
		endpoint:         config.Endpoint,
		externalEndpoint: config.ExternalEndpoint,
		accessKey:        config.AccessKey,
		secretKey:        config.SecretKey,
		region:           config.Region,
		providerName:     "RustFS",
	})
}

func (s RustFSStorage) GetBucketName() string {
	return s.BucketName
}

func (s RustFSStorage) PresignedGetObject(objectPath string, opts GetObjectOptions) (string, error) {
	var reqParams url.Values
	if opts.InlineContentType != "" {
		reqParams = url.Values{
			respContentDisposition: []string{"inline"},
			respContentType:        []string{opts.InlineContentType},
		}
	} else if opts.DownloadFilename != "" {
		reqParams = url.Values{
			respContentDisposition: []string{attachmentDisposition(opts.DownloadFilename)},
		}
	}

	presignedURL, err := s.signingClient.PresignedGetObject(
		context.Background(),
		s.BucketName,
		objectPath,
		c.UploadPolicyExpirationInMinutes*time.Minute,
		reqParams,
	)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (s RustFSStorage) PresignUpload(
	objectPath string,
	size int,
	metadata map[string]string,
) (PresignedUpload, error) {
	return presignS3Upload(s.storage, s.signingClient, s.BucketName, objectPath, size, metadata)
}

func (s RustFSStorage) SupportsMultipart() bool {
	return true
}

func (s RustFSStorage) ListObjectParts(path, uploadID string) ([]PartInfo, error) {
	return s3ListObjectParts(s.storage, s.BucketName, path, uploadID)
}

func (s RustFSStorage) CompleteMultipartUpload(path, uploadID string, parts []PartInfo, _ map[string]string) error {
	return s3CompleteMultipartUpload(s.storage, s.BucketName, path, uploadID, parts)
}

func (s RustFSStorage) AbortMultipartUpload(path, uploadID string) error {
	return s3AbortMultipartUpload(s.storage, s.BucketName, path, uploadID)
}

func (s RustFSStorage) StatObject(path string) (map[string]string, error) {
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

func (s RustFSStorage) ListObjects(prefix string, maxKeys int32) ([]string, error) {
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

func (s RustFSStorage) RemoveObject(path string) error {
	return s.storage.RemoveObject(
		context.Background(),
		s.BucketName,
		path,
		minio.RemoveObjectOptions{},
	)
}

func (s RustFSStorage) RemoveObjects(paths []string) error {
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, p := range paths {
			objectsCh <- minio.ObjectInfo{Key: p}
		}
	}()

	errorCh := s.storage.RemoveObjects(context.Background(), s.BucketName, objectsCh, minio.RemoveObjectsOptions{})

	for err := range errorCh {
		if err.Err != nil {
			zap.L().
				Error("Failed to delete object", zap.String("key", err.ObjectName), zap.Error(err.Err))
			return err.Err
		}
	}

	return nil
}

// EnsureMultipartCleanupPolicy is a no-op: MinIO and RustFS do not fully support
// the AbortIncompleteMultipartUpload lifecycle action.
// References:
// - https://github.com/minio/minio/issues/16120
// - https://github.com/minio/minio/issues/19115
func (s RustFSStorage) EnsureMultipartCleanupPolicy() error {
	return nil
}
