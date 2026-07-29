package storage

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	gcs "cloud.google.com/go/storage"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
)

type GCPStorage struct {
	BucketName   string
	storage      *gcs.Client
	authedClient *http.Client
}

func NewGCPStorage(bucketName string) IStorage {
	ctx := context.Background()

	client, err := gcs.NewClient(ctx)
	if err != nil {
		zap.L().Fatal("Failed to create storage client", zap.Error(err))
	}

	_, err = client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		zap.L().Fatal("Failed to connect to storage or bucket does not exist",
			zap.String("bucketName", bucketName),
			zap.Error(err))
	}

	authedClient, err := google.DefaultClient(ctx, gcs.ScopeReadWrite)
	if err != nil {
		zap.L().Fatal("Failed to create authenticated storage client", zap.Error(err))
	}

	return &GCPStorage{
		BucketName:   bucketName,
		storage:      client,
		authedClient: authedClient,
	}
}

func (g GCPStorage) GetBucketName() string {
	return g.BucketName
}

func (g GCPStorage) PresignUpload(
	objectPath string,
	size int,
	metadata map[string]string,
) (PresignedUpload, error) {
	expires := time.Now().Add(c.UploadPolicyExpirationInMinutes * time.Minute)

	headers := map[string]string{}
	for header, value := range map[string]string{
		"x-goog-meta-bucket-id": metadata["bucket_id"],
		"x-goog-meta-file-id":   metadata["file_id"],
		"x-goog-meta-user-id":   metadata["user_id"],
		"x-goog-meta-share-id":  metadata["share_id"],
	} {
		if value != "" {
			headers[header] = value
		}
	}

	if int64(size) <= c.MultipartPartSize {
		// GCS ignores x-goog-content-length
		headers["x-goog-content-length-range"] = fmt.Sprintf("%d,%d", int64(size), int64(size))

		opts := &gcs.SignedURLOptions{
			Method:  http.MethodPut,
			Expires: expires,
			Scheme:  gcs.SigningSchemeV4,
		}
		for key, value := range headers {
			opts.Headers = append(opts.Headers, key+":"+value)
		}

		signedURL, err := g.storage.Bucket(g.BucketName).SignedURL(objectPath, opts)
		if err != nil {
			return PresignedUpload{}, err
		}

		return PresignedUpload{Response: models.FileUploadResponse{
			Method: c.UploadMethodPut,
			Parts: []models.FilePartURL{
				{ID: 1, URL: signedURL, Size: int64(size), Headers: headers},
			},
		}}, nil
	}

	uploadID, err := g.initiateMultipartUpload(objectPath, headers)
	if err != nil {
		return PresignedUpload{}, err
	}

	partSize, partCount := ComputeMultipartLayout(int64(size))
	parts := make([]models.FilePartURL, 0, partCount)
	for partNumber := 1; partNumber <= partCount; partNumber++ {
		expected := ExpectedPartSize(int64(size), partSize, partNumber, partCount)
		rangeHeader := fmt.Sprintf("%d,%d", expected, expected)

		opts := &gcs.SignedURLOptions{
			Method:  http.MethodPut,
			Expires: expires,
			Scheme:  gcs.SigningSchemeV4,
			// GCS ignores x-goog-content-length
			Headers: []string{fmt.Sprintf("x-goog-content-length-range:%s", rangeHeader)},
			QueryParameters: url.Values{
				"uploadId":   []string{uploadID},
				"partNumber": []string{strconv.Itoa(partNumber)},
			},
		}

		signedURL, partErr := g.storage.Bucket(g.BucketName).SignedURL(objectPath, opts)
		if partErr != nil {
			if abortErr := g.AbortMultipartUpload(objectPath, uploadID); abortErr != nil {
				zap.L().Warn("Failed to abort multipart upload after part URL error", zap.Error(abortErr))
			}
			return PresignedUpload{}, partErr
		}
		parts = append(parts, models.FilePartURL{
			ID:   partNumber,
			URL:  signedURL,
			Size: expected,
			// GCS ignores x-goog-content-length
			Headers: map[string]string{"x-goog-content-length-range": rangeHeader},
		})
	}

	return PresignedUpload{
		Response: models.FileUploadResponse{Method: c.UploadMethodPut, Parts: parts},
		UploadID: uploadID,
		PartSize: partSize,
	}, nil
}

func (g GCPStorage) SupportsMultipart() bool {
	return true
}

func (g GCPStorage) objectURL(objectPath string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "storage.googleapis.com",
		Path:   fmt.Sprintf("/%s/%s", g.BucketName, objectPath),
	}
}

func (g GCPStorage) initiateMultipartUpload(objectPath string, metaHeaders map[string]string) (string, error) {
	u := g.objectURL(objectPath)
	u.RawQuery = "uploads"

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), nil)
	if err != nil {
		return "", err
	}
	for key, value := range metaHeaders {
		req.Header.Set(key, value)
	}

	resp, err := g.authedClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("initiate multipart upload failed: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		UploadID string `xml:"UploadId"`
	}
	if err = xml.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.UploadID == "" {
		return "", errors.New("initiate multipart upload: empty upload id")
	}

	return result.UploadID, nil
}

func (g GCPStorage) ListObjectParts(objectPath, uploadID string) ([]PartInfo, error) {
	var parts []PartInfo
	partNumberMarker := ""
	for {
		values := url.Values{"uploadId": {uploadID}}
		if partNumberMarker != "" {
			values.Set("part-number-marker", partNumberMarker)
		}
		u := g.objectURL(objectPath)
		u.RawQuery = values.Encode()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}

		resp, err := g.authedClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("list multipart parts failed: status %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			IsTruncated          bool   `xml:"IsTruncated"`
			NextPartNumberMarker string `xml:"NextPartNumberMarker"`
			Parts                []struct {
				PartNumber   int       `xml:"PartNumber"`
				ETag         string    `xml:"ETag"`
				Size         int64     `xml:"Size"`
				LastModified time.Time `xml:"LastModified"`
			} `xml:"Part"`
		}
		if err = xml.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		for _, part := range result.Parts {
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

func (g GCPStorage) CompleteMultipartUpload(
	objectPath, uploadID string, parts []PartInfo, _ map[string]string,
) error {
	type completePart struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	}
	payload := struct {
		XMLName xml.Name       `xml:"CompleteMultipartUpload"`
		Parts   []completePart `xml:"Part"`
	}{}
	for _, part := range parts {
		payload.Parts = append(payload.Parts, completePart{PartNumber: part.PartNumber, ETag: part.ETag})
	}

	body, err := xml.Marshal(payload)
	if err != nil {
		return err
	}

	u := g.objectURL(objectPath)
	u.RawQuery = url.Values{"uploadId": {uploadID}}.Encode()

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, u.String(), strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml")

	resp, err := g.authedClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("complete multipart upload failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (g GCPStorage) AbortMultipartUpload(objectPath, uploadID string) error {
	u := g.objectURL(objectPath)
	u.RawQuery = url.Values{"uploadId": {uploadID}}.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := g.authedClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("abort multipart upload failed: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (g GCPStorage) PresignedGetObject(objectPath string, opts GetObjectOptions) (string, error) {
	signOpts := &gcs.SignedURLOptions{
		Method:  http.MethodGet,
		Expires: time.Now().Add(c.UploadPolicyExpirationInMinutes * time.Minute),
	}
	if opts.InlineContentType != "" {
		signOpts.QueryParameters = url.Values{
			respContentDisposition: []string{"inline"},
			respContentType:        []string{opts.InlineContentType},
		}
	} else if opts.DownloadFilename != "" {
		signOpts.QueryParameters = url.Values{
			respContentDisposition: []string{attachmentDisposition(opts.DownloadFilename)},
		}
	}

	return g.storage.Bucket(g.BucketName).SignedURL(objectPath, signOpts)
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
	for _, p := range paths {
		if err := g.RemoveObject(p); err != nil {
			zap.L().Error("Failed to delete object", zap.String("key", p), zap.Error(err))
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

// IsTrashMarkerPath checks if a deletion event is for a trash marker.
// Patterns:
//   - trash/{bucket-id}/files/{file-id} -> buckets/{bucket-id}/{file-id}
//   - trash/{bucket-id}/folders/{folder-id} -> buckets/{bucket-id}/{folder-id}
func (g GCPStorage) IsTrashMarkerPath(path string) (bool, string) {
	if !strings.HasPrefix(path, trashPrefix) {
		return false, ""
	}

	remainder := strings.TrimPrefix(path, trashPrefix)
	parts := strings.SplitN(remainder, "/", 3)

	if len(parts) < 3 {
		return false, ""
	}

	bucketID := parts[0]
	resourceType := parts[1] // "files" or "folders"
	resourceID := parts[2]

	if resourceType != "files" && resourceType != "folders" {
		return false, ""
	}

	originalPath := bucketsPrefix + bucketID + "/" + resourceID
	return true, originalPath
}

// getTrashMarkerPath converts buckets/{bucket-id}/{id} to trash/{bucket-id}/files|folders/{id}.
func (g GCPStorage) getTrashMarkerPath(objectPath string, model interface{}) string {
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

	return path.Join(trashPrefix, bucketID, resourceType, resourceID)
}

func (g GCPStorage) MarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath, object)

	if _, ok := object.(models.File); ok {
		obj := g.storage.Bucket(g.BucketName).Object(objectPath)
		if _, err := obj.Attrs(ctx); err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)
	writer := markerObj.NewWriter(ctx)

	if _, err := writer.Write([]byte{}); err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	return nil
}

func (g GCPStorage) UnmarkAsTrashed(objectPath string, object interface{}) error {
	ctx := context.Background()
	markerPath := g.getTrashMarkerPath(objectPath, object)

	markerObj := g.storage.Bucket(g.BucketName).Object(markerPath)
	if err := markerObj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to remove marker: %w", err)
	}

	return nil
}

func (g GCPStorage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	ctx := context.Background()
	bucket := g.storage.Bucket(g.BucketName)

	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		zap.L().Error("Failed to get bucket attributes",
			zap.String("bucket", g.BucketName),
			zap.Error(err))
		return err
	}

	const trashRuleActionType = gcs.DeleteAction
	const multipartRuleActionType = gcs.AbortIncompleteMPUAction
	var existingTrashRuleIndex = -1
	var existingMultipartRuleIndex = -1

	if attrs.Lifecycle.Rules != nil {
		for i, rule := range attrs.Lifecycle.Rules {
			if rule.Action.Type == trashRuleActionType &&
				rule.Condition.MatchesPrefix != nil &&
				len(rule.Condition.MatchesPrefix) > 0 &&
				rule.Condition.MatchesPrefix[0] == trashPrefix {
				existingTrashRuleIndex = i

				if rule.Condition.AgeInDays == int64(retentionDays) {
					zap.L().Debug("Trash lifecycle policy already up-to-date",
						zap.String("bucket", g.BucketName),
						zap.Int("retentionDays", retentionDays))
				}
			}

			if rule.Action.Type == multipartRuleActionType &&
				rule.Condition.AgeInDays == 1 {
				existingMultipartRuleIndex = i
				zap.L().Debug("Multipart upload cleanup policy already up-to-date",
					zap.String("bucket", g.BucketName))
			}
		}
	}

	trashRule := gcs.LifecycleRule{
		Action: gcs.LifecycleAction{
			Type: trashRuleActionType,
		},
		Condition: gcs.LifecycleCondition{
			AgeInDays:     int64(retentionDays),
			MatchesPrefix: []string{trashPrefix},
		},
	}

	multipartRule := gcs.LifecycleRule{
		Action: gcs.LifecycleAction{
			Type: multipartRuleActionType,
		},
		Condition: gcs.LifecycleCondition{
			AgeInDays: 1,
		},
	}

	var newRules []gcs.LifecycleRule
	if attrs.Lifecycle.Rules != nil {
		for i, rule := range attrs.Lifecycle.Rules {
			if i != existingTrashRuleIndex && i != existingMultipartRuleIndex {
				newRules = append(newRules, rule)
			}
		}
	}
	newRules = append(newRules, trashRule, multipartRule)

	updateAttrs := gcs.BucketAttrsToUpdate{
		Lifecycle: &gcs.Lifecycle{
			Rules: newRules,
		},
	}

	if _, err = bucket.Update(ctx, updateAttrs); err != nil {
		zap.L().Error("Failed to update lifecycle policies",
			zap.String("bucket", g.BucketName),
			zap.Int("trashRetentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	zap.L().Info("Lifecycle policies configured",
		zap.String("bucket", g.BucketName),
		zap.Int("trashRetentionDays", retentionDays),
		zap.Int("multipartCleanupDays", 1))

	return nil
}
