package storage

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	c "api/internal/configuration"
	"api/internal/models"

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
		zap.L().
			Error("Failed to retrieve bucket.", zap.String("bucketName", bucketName), zap.Error(err))
	}

	presigner := s3.NewPresignClient(client)

	return AWSStorage{BucketName: bucketName, storage: client, presigner: presigner}
}

func (a AWSStorage) GetBucketName() string {
	return a.BucketName
}

func (a AWSStorage) PresignedGetObject(path string) (string, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	}

	resp, err := a.presigner.PresignGetObject(
		context.Background(),
		req,
		s3.WithPresignExpires(15*time.Minute),
	)
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (a AWSStorage) PresignedPostPolicy(
	path string,
	size int,
	metadata map[string]string,
) (string, map[string]string, error) {
	req := &s3.PutObjectInput{
		Bucket:        aws.String(a.BucketName),
		Key:           aws.String(path),
		ContentLength: aws.Int64(int64(size)),
		Expires: aws.Time(
			time.Now().UTC().Add(c.UploadPolicyExpirationInMinutes * time.Minute),
		),
	}

	// FIXME(YLB): Workaround to sign the metadata
	// https://github.com/aws/aws-sdk-go-v2/issues/3119
	metaFields := []string{"bucket_id", "file_id", "user_id"}

	var conditions []interface{}
	for _, field := range metaFields {
		conditions = append(conditions, map[string]string{
			"x-amz-meta-" + field: metadata[field],
		})
	}

	presignedPost, err := a.presigner.PresignPostObject(
		context.Background(),
		req,
		func(opts *s3.PresignPostOptions) {
			opts.Conditions = conditions
		},
	)
	if err != nil {
		return "", nil, err
	}

	for _, field := range metaFields {
		key := "x-amz-meta-" + field
		presignedPost.Values[key] = metadata[field]
	}

	return presignedPost.URL, presignedPost.Values, nil
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

		for j, path := range batch {
			objects[j] = types.ObjectIdentifier{
				Key: aws.String(path),
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

func (a AWSStorage) SetObjectTags(path string, tagMap map[string]string) error {
	var tagSet []types.Tag
	for key, value := range tagMap {
		tagSet = append(tagSet, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err := a.storage.PutObjectTagging(context.Background(), &s3.PutObjectTaggingInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
		Tagging: &types.Tagging{
			TagSet: tagSet,
		},
	})
	return err
}

func (a AWSStorage) GetObjectTags(path string) (map[string]string, error) {
	tagsOutput, err := a.storage.GetObjectTagging(context.Background(), &s3.GetObjectTaggingInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]string)
	for _, tag := range tagsOutput.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tagMap[*tag.Key] = *tag.Value
		}
	}
	return tagMap, nil
}

func (a AWSStorage) RemoveObjectTags(path string, tagsToRemove []string) error {
	tagsOutput, err := a.storage.GetObjectTagging(context.Background(), &s3.GetObjectTaggingInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return err
	}

	removeSet := make(map[string]bool)
	for _, key := range tagsToRemove {
		removeSet[key] = true
	}

	var filteredTags []types.Tag
	for _, tag := range tagsOutput.TagSet {
		if tag.Key != nil && !removeSet[*tag.Key] {
			filteredTags = append(filteredTags, tag)
		}
	}

	_, err = a.storage.PutObjectTagging(context.Background(), &s3.PutObjectTaggingInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
		Tagging: &types.Tagging{
			TagSet: filteredTags,
		},
	})
	return err
}

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Pattern: trash/{bucket-id}/{rest} -> buckets/{bucket-id}/{rest}.
func (a AWSStorage) IsTrashMarkerPath(path string) (bool, string) {
	if strings.HasPrefix(path, trashPrefix) {
		originalPath := bucketsPrefix + strings.TrimPrefix(path, trashPrefix)
		return true, originalPath
	}

	return false, ""
}

// getTrashMarkerPath converts buckets/{id}/path to trash/{id}/path.
func (a AWSStorage) getTrashMarkerPath(objectPath string) string {
	return strings.Replace(objectPath, bucketsPrefix, trashPrefix, 1)
}

func (a AWSStorage) MarkFileAsTrashed(objectPath string, metadata models.TrashMetadata) error {
	ctx := context.Background()
	markerPath := a.getTrashMarkerPath(objectPath)

	if !metadata.IsFolder {
		_, err := a.storage.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(a.BucketName),
			Key:    aws.String(objectPath),
		})
		if err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	// Create empty marker object to trigger lifecycle policy deletion
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

func (a AWSStorage) UnmarkFileAsTrashed(objectPath string) error {
	ctx := context.Background()
	markerPath := a.getTrashMarkerPath(objectPath)

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

	// Validate retentionDays fits in int32 to prevent overflow
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
