package storage

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// presignS3Upload builds a presigned upload for any minio-backed S3 provider
// (MinIO, RustFS, and generic S3-compatible providers). Small files get a
// single presigned PUT; larger files get a multipart upload with one presigned
// PUT per part. storage runs the multipart control-plane calls (internal
// endpoint); signingClient signs the URLs the client uses (external endpoint).
func presignS3Upload(
	storage, signingClient *minio.Client,
	bucketName, objectPath string,
	size int,
	metadata map[string]string,
) (PresignedUpload, error) {
	ctx := context.Background()
	userMetadata := map[string]string{
		"Bucket-Id": metadata["bucket_id"],
		"File-Id":   metadata["file_id"],
		"User-Id":   metadata["user_id"],
		"Share-Id":  metadata["share_id"],
	}

	if int64(size) <= c.MultipartPartSize {
		metaHeaders := http.Header{}
		for key, value := range userMetadata {
			metaHeaders.Set("X-Amz-Meta-"+key, value)
		}

		signHeaders := metaHeaders.Clone()
		signHeaders.Set("Content-Length", strconv.FormatInt(int64(size), 10))

		presignedURL, err := signingClient.PresignHeader(
			ctx, http.MethodPut, bucketName, objectPath,
			c.UploadPolicyExpirationInMinutes*time.Minute, nil, signHeaders,
		)
		if err != nil {
			return PresignedUpload{}, err
		}

		clientHeaders := make(map[string]string, len(metaHeaders))
		for key := range metaHeaders {
			clientHeaders[key] = metaHeaders.Get(key)
		}

		return PresignedUpload{Response: models.FileUploadResponse{
			Method: c.UploadMethodPut,
			Parts: []models.FilePartURL{
				{PartNumber: 1, URL: presignedURL.String(), Size: int64(size), Headers: clientHeaders},
			},
		}}, nil
	}

	core := minio.Core{Client: storage}
	uploadID, err := core.NewMultipartUpload(
		ctx, bucketName, objectPath, minio.PutObjectOptions{UserMetadata: userMetadata},
	)
	if err != nil {
		return PresignedUpload{}, err
	}

	partSize, partCount := ComputeMultipartLayout(int64(size))
	parts := make([]models.FilePartURL, 0, partCount)
	for partNumber := 1; partNumber <= partCount; partNumber++ {
		expected := ExpectedPartSize(int64(size), partSize, partNumber, partCount)

		reqParams := url.Values{
			"uploadId":   []string{uploadID},
			"partNumber": []string{strconv.Itoa(partNumber)},
		}
		extraHeaders := http.Header{}
		extraHeaders.Set("Content-Length", strconv.FormatInt(expected, 10))

		presignedURL, partErr := signingClient.PresignHeader(
			ctx, http.MethodPut, bucketName, objectPath,
			c.UploadPolicyExpirationInMinutes*time.Minute, reqParams, extraHeaders,
		)
		if partErr != nil {
			if abortErr := s3AbortMultipartUpload(storage, bucketName, objectPath, uploadID); abortErr != nil {
				zap.L().Warn("Failed to abort multipart upload after part URL error", zap.Error(abortErr))
			}
			return PresignedUpload{}, partErr
		}
		parts = append(parts, models.FilePartURL{PartNumber: partNumber, URL: presignedURL.String(), Size: expected})
	}

	return PresignedUpload{
		Response: models.FileUploadResponse{Method: c.UploadMethodPut, Parts: parts},
		UploadID: uploadID,
		PartSize: partSize,
	}, nil
}

func s3ListObjectParts(storage *minio.Client, bucketName, path, uploadID string) ([]PartInfo, error) {
	core := minio.Core{Client: storage}

	var parts []PartInfo
	partNumberMarker := 0
	for {
		result, err := core.ListObjectParts(context.Background(), bucketName, path, uploadID, partNumberMarker, 1000)
		if err != nil {
			return nil, err
		}

		for _, part := range result.ObjectParts {
			parts = append(parts, PartInfo{
				PartNumber:   part.PartNumber,
				Size:         part.Size,
				ETag:         part.ETag,
				LastModified: part.LastModified,
			})
		}

		if !result.IsTruncated {
			break
		}
		partNumberMarker = result.NextPartNumberMarker
	}

	return parts, nil
}

func s3CompleteMultipartUpload(storage *minio.Client, bucketName, path, uploadID string, parts []PartInfo) error {
	core := minio.Core{Client: storage}

	completeParts := make([]minio.CompletePart, len(parts))
	for i, part := range parts {
		completeParts[i] = minio.CompletePart{PartNumber: part.PartNumber, ETag: part.ETag}
	}

	_, err := core.CompleteMultipartUpload(
		context.Background(), bucketName, path, uploadID, completeParts, minio.PutObjectOptions{},
	)
	return err
}

func s3AbortMultipartUpload(storage *minio.Client, bucketName, path, uploadID string) error {
	core := minio.Core{Client: storage}

	err := core.AbortMultipartUpload(context.Background(), bucketName, path, uploadID)
	if err != nil && minio.ToErrorResponse(err).Code == "NoSuchUpload" {
		return nil
	}
	return err
}
