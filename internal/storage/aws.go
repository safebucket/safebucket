package storage

import (
	"context"
	"time"

	c "api/internal/configuration"

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
