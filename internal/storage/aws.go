package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path"
	"strings"
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

		_, err := a.storage.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
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
	}

	return nil
}

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Patterns:
//   - trash/{bucket-id}/files/{file-id} -> buckets/{bucket-id}/{file-id}
//   - trash/{bucket-id}/folders/{folder-id} -> buckets/{bucket-id}/{folder-id}
func (a AWSStorage) IsTrashMarkerPath(path string) (bool, string) {
	if !strings.HasPrefix(path, trashPrefix) {
		return false, ""
	}

	remainder := strings.TrimPrefix(path, trashPrefix)
	parts := strings.SplitN(remainder, "/", 3)

	if len(parts) < 3 {
		return false, ""
	}

	bucketID := parts[0]
	resourceType := parts[1]
	resourceID := parts[2]

	if resourceType != "files" && resourceType != "folders" {
		return false, ""
	}

	originalPath := bucketsPrefix + bucketID + "/" + resourceID
	return true, originalPath
}

// getTrashMarkerPath converts buckets/{bucket-id}/{id} to trash/{bucket-id}/files|folders/{id}.
func (a AWSStorage) getTrashMarkerPath(objectPath string, model interface{}) string {
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

func (a AWSStorage) MarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := a.getTrashMarkerPath(objectPath, object)

	if _, ok := object.(models.File); ok {
		_, err := a.storage.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(a.BucketName),
			Key:    aws.String(objectPath),
		})
		if err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	reader := bytes.NewReader([]byte{})
	_, err := a.storage.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(markerPath),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	return nil
}

func (a AWSStorage) UnmarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := a.getTrashMarkerPath(objectPath, object)

	_, err := a.storage.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(markerPath),
	})
	if err != nil {
		return fmt.Errorf("failed to remove marker: %w", err)
	}

	return nil
}

// isTrashLifecycleRuleUpToDate checks if an existing lifecycle rule matches the desired trash retention configuration.
func isTrashLifecycleRuleUpToDate(rule types.LifecycleRule, retentionDays int) bool {
	if rule.Expiration == nil || rule.Expiration.Days == nil {
		return false
	}
	if *rule.Expiration.Days != int32(retentionDays) { //nolint:gosec // validated in EnsureTrashLifecyclePolicy
		return false
	}
	if rule.Filter == nil || rule.Filter.Prefix == nil {
		return false
	}
	if *rule.Filter.Prefix != trashPrefix {
		return false
	}

	return true
}

// processExistingLifecycleRules processes existing lifecycle rules and returns the updated rules list.
func (a AWSStorage) processExistingLifecycleRules(
	existingConfig *s3.GetBucketLifecycleConfigurationOutput,
	err error,
	trashRuleID, multipartRuleID string,
	trashRule, multipartRule types.LifecycleRule,
	retentionDays int,
) ([]types.LifecycleRule, bool, bool) {
	var rules []types.LifecycleRule
	trashRuleFound := false
	multipartRuleFound := false

	if err != nil || existingConfig == nil {
		// No existing config - return new rules with flags set to true
		// to prevent duplicate addition in the caller
		return []types.LifecycleRule{trashRule, multipartRule}, true, true
	}

	for _, rule := range existingConfig.Rules {
		if rule.ID == nil {
			rules = append(rules, rule)
			continue
		}

		switch *rule.ID {
		case trashRuleID:
			if isTrashLifecycleRuleUpToDate(rule, retentionDays) {
				zap.L().Debug("Trash lifecycle policy already up-to-date",
					zap.String("bucket", a.BucketName),
					zap.Int("retentionDays", retentionDays))
			}
			trashRuleFound = true
			rules = append(rules, trashRule)

		case multipartRuleID:
			multipartRuleFound = true
			if rule.AbortIncompleteMultipartUpload != nil &&
				rule.AbortIncompleteMultipartUpload.DaysAfterInitiation != nil &&
				*rule.AbortIncompleteMultipartUpload.DaysAfterInitiation == 1 {
				zap.L().Debug("Multipart upload cleanup policy already up-to-date",
					zap.String("bucket", a.BucketName))
				rules = append(rules, rule)
			} else {
				rules = append(rules, multipartRule)
			}

		default:
			rules = append(rules, rule)
		}
	}

	return rules, trashRuleFound, multipartRuleFound
}

func (a AWSStorage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	const trashRuleID = "safebucket-trash-retention"
	const multipartRuleID = "safebucket-abort-incomplete-multipart"

	if retentionDays < 0 || retentionDays > math.MaxInt32 {
		return fmt.Errorf("retentionDays %d is out of valid range (0-%d)", retentionDays, math.MaxInt32)
	}

	ctx := context.Background()

	existingConfig, err := a.storage.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(a.BucketName),
	})

	trashRule := types.LifecycleRule{
		ID:     aws.String(trashRuleID),
		Status: types.ExpirationStatusEnabled,
		Filter: &types.LifecycleRuleFilter{
			Prefix: aws.String(trashPrefix),
		},
		Expiration: &types.LifecycleExpiration{
			Days: aws.Int32(int32(retentionDays)),
		},
	}

	multipartRule := types.LifecycleRule{
		ID:     aws.String(multipartRuleID),
		Status: types.ExpirationStatusEnabled,
		Filter: &types.LifecycleRuleFilter{
			Prefix: aws.String(""),
		},
		AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: aws.Int32(1),
		},
	}

	rules, trashRuleFound, multipartRuleFound := a.processExistingLifecycleRules(
		existingConfig,
		err,
		trashRuleID,
		multipartRuleID,
		trashRule,
		multipartRule,
		retentionDays,
	)

	if !trashRuleFound {
		rules = append(rules, trashRule)
	}
	if !multipartRuleFound {
		rules = append(rules, multipartRule)
	}

	{
		_, err = a.storage.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String(a.BucketName),
			LifecycleConfiguration: &types.BucketLifecycleConfiguration{
				Rules: rules,
			},
		})
		if err != nil {
			zap.L().Error("Failed to set lifecycle policies",
				zap.String("bucket", a.BucketName),
				zap.Int("trashRetentionDays", retentionDays),
				zap.Error(err))
			return err
		}

		zap.L().Info("Lifecycle policies configured",
			zap.String("bucket", a.BucketName),
			zap.Int("trashRetentionDays", retentionDays),
			zap.Int("multipartCleanupDays", 1))
		return nil
	}
}
