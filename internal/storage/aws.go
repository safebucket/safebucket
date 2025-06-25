package storage

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"time"
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
		zap.L().Error("Failed to retrieve bucket.", zap.String("bucketName", bucketName), zap.Error(err))
	}

	presigner := s3.NewPresignClient(client)

	return AWSStorage{BucketName: bucketName, storage: client, presigner: presigner}
}

func (a AWSStorage) PresignedGetObject(path string) (string, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	}

	resp, err := a.presigner.PresignGetObject(context.Background(), req, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (a AWSStorage) PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error) {
	req := &s3.PutObjectInput{
		Bucket:        aws.String(a.BucketName),
		Key:           aws.String(path),
		ContentLength: aws.Int64(int64(size)),
		Expires:       aws.Time(time.Now().UTC().Add(15 * time.Minute)),
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

	presignedPost, err := a.presigner.PresignPostObject(context.Background(), req, func(opts *s3.PresignPostOptions) {
		opts.Conditions = conditions
	})

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

func (a AWSStorage) RemoveObject(path string) error {
	_, err := a.storage.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(path),
	})
	return err
}
