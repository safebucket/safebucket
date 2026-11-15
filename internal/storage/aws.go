package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

type AWSStorage struct {
	BucketName string
	storage    *s3.Client
	presigner  *s3.PresignClient
}

func NewAWSStorage(bucketName string) IStorage {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		zap.L().Fatal("Unable to load SDK config.", zap.Error(err))
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		zap.L().Fatal("Failed to connect to storage or bucket does not exist",
			zap.String("bucketName", bucketName),
			zap.Error(err))
	}

	presigner := s3.NewPresignClient(client)

	return AWSStorage{BucketName: bucketName, storage: client, presigner: presigner}
}

func (a AWSStorage) GetBucketName() string {
	return a.BucketName
}

func (a AWSStorage) PresignUpload(
	objectPath string,
	size int,
	metadata map[string]string,
) (PresignedUpload, error) {
	ctx := context.Background()
	expires := c.UploadPolicyExpirationInMinutes * time.Minute

	if int64(size) <= c.MultipartPartSize {
		presigned, err := a.presigner.PresignPutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket:        aws.String(a.BucketName),
				Key:           aws.String(objectPath),
				ContentLength: aws.Int64(int64(size)),
				Metadata:      metadata,
			},
			s3.WithPresignExpires(expires),
		)
		if err != nil {
			return PresignedUpload{}, err
		}

		clientHeaders := make(map[string]string, len(presigned.SignedHeader))
		for key := range presigned.SignedHeader {
			switch http.CanonicalHeaderKey(key) {
			case "Content-Length", "Host":
				continue
			}

			clientHeaders[key] = presigned.SignedHeader.Get(key)
		}

		return PresignedUpload{Response: models.FileUploadResponse{
			Method: c.UploadMethodPut,
			Parts: []models.FilePartURL{
				{
					ID:      1,
					URL:     presigned.URL,
					Size:    int64(size),
					Headers: clientHeaders,
				},
			},
		}}, nil
	}

	created, err := a.storage.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:   aws.String(a.BucketName),
		Key:      aws.String(objectPath),
		Metadata: metadata,
	})
	if err != nil {
		return PresignedUpload{}, err
	}
	uploadID := aws.ToString(created.UploadId)

	partSize, partCount := ComputeMultipartLayout(int64(size))
	parts := make([]models.FilePartURL, 0, partCount)
	for partNumber := 1; partNumber <= partCount; partNumber++ {
		expected := ExpectedPartSize(int64(size), partSize, partNumber, partCount)

		presigned, partErr := a.presigner.PresignUploadPart(
			ctx,
			&s3.UploadPartInput{
				Bucket:        aws.String(a.BucketName),
				Key:           aws.String(objectPath),
				UploadId:      aws.String(uploadID),
				PartNumber:    aws.Int32(int32(partNumber)),
				ContentLength: aws.Int64(partSize),
			},
			s3.WithPresignExpires(expires),
		)
		if partErr != nil {
			if abortErr := a.AbortMultipartUpload(objectPath, uploadID); abortErr != nil {
				zap.L().Warn("Failed to abort multipart upload after part URL error", zap.Error(abortErr))
			}
			return PresignedUpload{}, partErr
		}
		parts = append(parts, models.FilePartURL{ID: partNumber, URL: presigned.URL, Size: expected})
	}

	return PresignedUpload{
		Response: models.FileUploadResponse{Method: c.UploadMethodPut, Parts: parts},
		UploadID: uploadID,
		PartSize: partSize,
	}, nil
}

func (a AWSStorage) SupportsMultipart() bool {
	return true
}

func (a AWSStorage) ListObjectParts(objectPath, uploadID string) ([]PartInfo, error) {
	ctx := context.Background()

	var parts []PartInfo
	var partNumberMarker *string
	for {
		result, err := a.storage.ListParts(ctx, &s3.ListPartsInput{
			Bucket:           aws.String(a.BucketName),
			Key:              aws.String(objectPath),
			UploadId:         aws.String(uploadID),
			PartNumberMarker: partNumberMarker,
		})
		if err != nil {
			return nil, err
		}

		for _, part := range result.Parts {
			parts = append(parts, PartInfo{
				PartNumber:   int(aws.ToInt32(part.PartNumber)),
				Size:         aws.ToInt64(part.Size),
				ETag:         aws.ToString(part.ETag),
				LastModified: aws.ToTime(part.LastModified),
			})
		}

		if !aws.ToBool(result.IsTruncated) {
			break
		}
		partNumberMarker = result.NextPartNumberMarker
	}

	return parts, nil
}

func (a AWSStorage) CompleteMultipartUpload(
	objectPath, uploadID string, parts []PartInfo, _ map[string]string,
) error {
	completeParts := make([]types.CompletedPart, len(parts))
	for i, part := range parts {
		partNumber := int32(part.PartNumber) //nolint:gosec // bounded by MultipartMaxParts
		completeParts[i] = types.CompletedPart{
			PartNumber: aws.Int32(partNumber),
			ETag:       aws.String(part.ETag),
		}
	}

	_, err := a.storage.CompleteMultipartUpload(context.Background(), &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(a.BucketName),
		Key:             aws.String(objectPath),
		UploadId:        aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completeParts},
	})
	return err
}

func (a AWSStorage) AbortMultipartUpload(objectPath, uploadID string) error {
	_, err := a.storage.AbortMultipartUpload(context.Background(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(a.BucketName),
		Key:      aws.String(objectPath),
		UploadId: aws.String(uploadID),
	})

	if _, ok := errors.AsType[*types.NoSuchUpload](err); ok {
		return nil
	}
	return err
}

func (a AWSStorage) PresignedGetObject(objectPath string, opts GetObjectOptions) (string, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(objectPath),
	}
	if opts.InlineContentType != "" {
		req.ResponseContentDisposition = aws.String("inline")
		req.ResponseContentType = aws.String(opts.InlineContentType)
	} else if opts.DownloadFilename != "" {
		req.ResponseContentDisposition = aws.String(attachmentDisposition(opts.DownloadFilename))
	}

	resp, err := a.presigner.PresignGetObject(
		context.Background(),
		req,
		s3.WithPresignExpires(c.UploadPolicyExpirationInMinutes*time.Minute),
	)
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (a AWSStorage) StatObject(path string) (map[string]string, error) {
	file, err := a.storage.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	return file.Metadata, err
}

func (a AWSStorage) ListObjects(prefix string, maxKeys int32) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(a.BucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	var objects []string

	result, err := a.storage.ListObjectsV2(context.Background(), input)
	if err != nil {
		return nil, err
	}

	for _, obj := range result.Contents {
		objects = append(objects, *obj.Key)
	}

	return objects, nil
}

func (a AWSStorage) RemoveObject(path string) error {
	_, err := a.storage.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	})
	return err
}

func (a AWSStorage) RemoveObjects(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// AWS S3 batch delete supports up to 1000 objects per request
	const batchSize = 1000

	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}

		batch := paths[i:end]
		objects := make([]types.ObjectIdentifier, len(batch))

		for j, p := range batch {
			objects[j] = types.ObjectIdentifier{
				Key: aws.String(p),
			}
		}

		output, err := a.storage.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
			Bucket: aws.String(a.BucketName),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			zap.L().
				Error("Failed to delete objects batch", zap.Int("batch_start", i), zap.Error(err))
			return err
		}
		if len(output.Errors) > 0 {
			failure := output.Errors[0]
			return fmt.Errorf("delete object %s: %s: %s",
				aws.ToString(failure.Key), aws.ToString(failure.Code), aws.ToString(failure.Message))
		}
	}

	return nil
}

func (a AWSStorage) EnsureMultipartCleanupPolicy() error {
	const multipartRuleID = "safebucket-abort-incomplete-multipart"

	ctx := context.Background()

	var rules []types.LifecycleRule
	existing, err := a.storage.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(a.BucketName),
	})
	if err == nil && existing != nil {
		for _, rule := range existing.Rules {
			if rule.ID == nil || *rule.ID != multipartRuleID {
				rules = append(rules, rule)
			}
		}
	}

	rules = append(rules, types.LifecycleRule{
		ID:     aws.String(multipartRuleID),
		Status: types.ExpirationStatusEnabled,
		Filter: &types.LifecycleRuleFilter{
			Prefix: aws.String(""),
		},
		AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: aws.Int32(1),
		},
	})

	_, err = a.storage.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket:                 aws.String(a.BucketName),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{Rules: rules},
	})
	if err != nil {
		zap.L().Error("Failed to set lifecycle policies", zap.String("bucket", a.BucketName), zap.Error(err))
		return err
	}

	zap.L().Info("Lifecycle policies configured",
		zap.String("bucket", a.BucketName),
		zap.Int("multipartCleanupDays", 1))
	return nil
}
