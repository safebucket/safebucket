package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const azureTrashLifecycleRuleName = "safebucket-trash-retention"

const azureUploadTokenPrefix = "azure:"

type AzureStorage struct {
	containerName  string
	accountName    string
	subscriptionID string
	resourceGroup  string
	endpoint       string
	container      *container.Client
	cred           *blob.SharedKeyCredential
	signer         *azureSharedKeySigner
	armCred        azcore.TokenCredential
}

func NewAzureStorage(config *models.AzureConfiguration) IStorage {
	accountKey, armCred := resolveAzureCredentials(config)

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", config.AccountName)
	}
	containerURL := strings.TrimRight(endpoint, "/") + "/" + config.ContainerName

	cred, err := blob.NewSharedKeyCredential(config.AccountName, accountKey)
	if err != nil {
		zap.L().Fatal("Failed to create Azure shared key credential", zap.Error(err))
	}

	containerClient, err := container.NewClientWithSharedKeyCredential(containerURL, cred, nil)
	if err != nil {
		zap.L().Fatal("Failed to create Azure container client", zap.Error(err))
	}

	if _, err = containerClient.GetProperties(context.Background(), nil); err != nil {
		zap.L().Fatal("Failed to connect to storage or container does not exist",
			zap.String("container", config.ContainerName), zap.Error(err))
	}

	signer, err := newAzureSharedKeySigner(config.AccountName, accountKey)
	if err != nil {
		zap.L().Fatal("Failed to create Azure shared key signer", zap.Error(err))
	}

	return &AzureStorage{
		containerName:  config.ContainerName,
		accountName:    config.AccountName,
		subscriptionID: config.SubscriptionID,
		resourceGroup:  config.ResourceGroup,
		endpoint:       endpoint,
		container:      containerClient,
		cred:           cred,
		signer:         signer,
		armCred:        armCred,
	}
}

func (a *AzureStorage) GetBucketName() string {
	return a.containerName
}

func (a *AzureStorage) blockBlobClient(objectPath string) *blockblob.Client {
	return a.container.NewBlockBlobClient(objectPath)
}

func (a *AzureStorage) blobClient(objectPath string) *blob.Client {
	return a.container.NewBlobClient(objectPath)
}

func (a *AzureStorage) blobURL(objectPath string) string {
	segments := strings.Split(objectPath, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.TrimRight(a.endpoint, "/") + "/" + a.containerName + "/" + strings.Join(segments, "/")
}

func (a *AzureStorage) PresignUpload(
	objectPath string,
	size int,
	metadata map[string]string,
) (PresignedUpload, error) {
	if int64(size) <= c.MultipartPartSize {
		return a.presignSinglePut(objectPath, size, metadata), nil
	}
	return a.presignMultipart(objectPath, size), nil
}

func (a *AzureStorage) presignSinglePut(objectPath string, size int, metadata map[string]string) PresignedUpload {
	extraHeaders := azureMetadataHeaders(metadata)
	extraHeaders[azureHeaderBlobType] = azureBlobTypeBlock

	const contentType = "application/octet-stream"
	headers := a.signer.headersForPut(a.containerName, objectPath, url.Values{}, int64(size), contentType, extraHeaders)

	return PresignedUpload{Response: models.FileUploadResponse{
		Method: c.UploadMethodPut,
		Parts: []models.FilePartURL{
			{PartNumber: 1, URL: a.blobURL(objectPath), Size: int64(size), Headers: headers},
		},
	}}
}

func (a *AzureStorage) presignMultipart(objectPath string, size int) PresignedUpload {
	partSize, partCount := ComputeMultipartLayout(int64(size))

	parts := make([]models.FilePartURL, 0, partCount)
	for partNumber := 1; partNumber <= partCount; partNumber++ {
		expected := ExpectedPartSize(int64(size), partSize, partNumber, partCount)

		query := url.Values{"comp": {"block"}, "blockid": {azureBlockID(partNumber)}}
		headers := a.signer.headersForPut(a.containerName, objectPath, query, expected, "", nil)

		parts = append(parts, models.FilePartURL{
			PartNumber: partNumber,
			URL:        a.blobURL(objectPath) + "?" + query.Encode(),
			Size:       expected,
			Headers:    headers,
		})
	}

	return PresignedUpload{
		Response: models.FileUploadResponse{Method: c.UploadMethodPut, Parts: parts},
		UploadID: azureNewUploadToken(),
		PartSize: partSize,
	}
}

func azureMetadataHeaders(metadata map[string]string) map[string]string {
	headers := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if value != "" {
			headers[azureMetaHeaderPfx+key] = value
		}
	}

	return headers
}

func azureNewUploadToken() string {
	return azureUploadTokenPrefix + uuid.NewString()
}

func azureBucketAndFileFromPath(objectPath string) (string, string) {
	remainder := strings.TrimPrefix(objectPath, bucketsPrefix)
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func azureCommitMetadata(objectPath string, metadata map[string]string) map[string]*string {
	bucketID, fileID := azureBucketAndFileFromPath(objectPath)

	values := map[string]string{"bucket_id": bucketID, "file_id": fileID}
	for key, value := range metadata {
		values[key] = value
	}

	result := map[string]*string{}
	for key, value := range values {
		if value != "" {
			result[key] = to.Ptr(value)
		}
	}

	return result
}

func (a *AzureStorage) SupportsMultipart() bool {
	return true
}

func (a *AzureStorage) ListObjectParts(objectPath, _ string) ([]PartInfo, error) {
	resp, err := a.blockBlobClient(objectPath).
		GetBlockList(context.Background(), blockblob.BlockListTypeUncommitted, nil)
	if err != nil {
		return nil, err
	}

	parts := make([]PartInfo, 0, len(resp.UncommittedBlocks))
	for _, block := range resp.UncommittedBlocks {
		if block.Name == nil || block.Size == nil {
			continue
		}
		partNumber, blockErr := azurePartNumberFromBlockID(*block.Name)
		if blockErr != nil {
			return nil, blockErr
		}
		parts = append(parts, PartInfo{PartNumber: partNumber, Size: *block.Size})
	}

	return parts, nil
}

func (a *AzureStorage) CompleteMultipartUpload(
	objectPath, _ string, parts []PartInfo, metadata map[string]string,
) error {
	sorted := append([]PartInfo(nil), parts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PartNumber < sorted[j].PartNumber })

	blockIDs := make([]string, len(sorted))
	var total int64
	for i, part := range sorted {
		blockIDs[i] = azureBlockID(part.PartNumber)
		total += part.Size
	}

	commitMetadata := azureCommitMetadata(objectPath, metadata)

	ctx := context.Background()

	if _, commitErr := a.blockBlobClient(objectPath).CommitBlockList(
		ctx, blockIDs, &blockblob.CommitBlockListOptions{Metadata: commitMetadata},
	); commitErr != nil {
		return fmt.Errorf("commit block list: %w", commitErr)
	}

	props, propsErr := a.blockBlobClient(objectPath).GetProperties(ctx, nil)
	if propsErr != nil {
		return fmt.Errorf("get blob properties after commit: %w", propsErr)
	}
	if props.ContentLength == nil || *props.ContentLength != total {
		return fmt.Errorf("%w: committed blob size does not match staged parts", ErrMultipartPartMismatch)
	}

	return nil
}

func (a *AzureStorage) AbortMultipartUpload(_, _ string) error {
	return nil
}

func (a *AzureStorage) PresignedGetObject(objectPath string, opts GetObjectOptions) (string, error) {
	values := sas.BlobSignatureValues{
		ExpiryTime:    time.Now().UTC().Add(c.UploadPolicyExpirationInMinutes * time.Minute),
		Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		ContainerName: a.containerName,
		BlobName:      objectPath,
	}

	if opts.InlineContentType != "" {
		values.ContentDisposition = "inline"
		values.ContentType = opts.InlineContentType
	} else if opts.DownloadFilename != "" {
		values.ContentDisposition = attachmentDisposition(opts.DownloadFilename)
	}

	qp, err := values.SignWithSharedKey(a.cred)
	if err != nil {
		return "", err
	}

	return a.blobURL(objectPath) + "?" + qp.Encode(), nil
}

func (a *AzureStorage) StatObject(objectPath string) (map[string]string, error) {
	props, err := a.blobClient(objectPath).GetProperties(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string, len(props.Metadata))
	for k, v := range props.Metadata {
		if v != nil {
			metadata[strings.ToLower(k)] = *v
		}
	}

	return metadata, nil
}

func (a *AzureStorage) ListObjects(prefix string, maxKeys int32) ([]string, error) {
	var objects []string

	pager := a.container.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{Prefix: &prefix})
	for pager.More() {
		page, pageErr := pager.NextPage(context.Background())
		if pageErr != nil {
			return nil, pageErr
		}
		if page.Segment == nil {
			continue
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			objects = append(objects, *item.Name)
			if maxKeys > 0 && len(objects) >= int(maxKeys) {
				return objects, nil
			}
		}
	}

	return objects, nil
}

func (a *AzureStorage) RemoveObject(objectPath string) error {
	_, err := a.blobClient(objectPath).Delete(context.Background(), nil)
	return err
}

func (a *AzureStorage) RemoveObjects(paths []string) error {
	for _, p := range paths {
		if err := a.RemoveObject(p); err != nil {
			zap.L().Error("Failed to delete object", zap.String("key", p), zap.Error(err))
			return err
		}
	}
	return nil
}

func (a *AzureStorage) IsTrashMarkerPath(markerPath string) (bool, string) {
	if !strings.HasPrefix(markerPath, trashPrefix) {
		return false, ""
	}

	remainder := strings.TrimPrefix(markerPath, trashPrefix)
	parts := strings.SplitN(remainder, "/", 3)
	if len(parts) < 3 {
		return false, ""
	}

	bucketID, resourceType, resourceID := parts[0], parts[1], parts[2]
	if resourceType != folderPath && resourceType != filePath {
		return false, ""
	}

	return true, bucketsPrefix + bucketID + "/" + resourceID
}

func (a *AzureStorage) getTrashMarkerPath(objectPath string, model interface{}) string {
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

	return path.Join(trashPrefix, parts[0], resourceType, parts[1])
}

func (a *AzureStorage) MarkAsTrashed(objectPath string, object interface{}) error {
	markerPath := a.getTrashMarkerPath(objectPath, object)
	ctx := context.Background()

	if _, ok := object.(models.File); ok {
		if _, err := a.blobClient(objectPath).GetProperties(ctx, nil); err != nil {
			return fmt.Errorf("object does not exist and can't be trashed: %w", err)
		}
	}

	_, err := a.blockBlobClient(markerPath).Upload(ctx, streaming.NopCloser(bytes.NewReader(nil)), nil)
	if err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}
	return nil
}

func (a *AzureStorage) UnmarkAsTrashed(objectPath string, object interface{}) error {
	markerPath := a.getTrashMarkerPath(objectPath, object)

	if _, err := a.blobClient(markerPath).Delete(context.Background(), nil); err != nil {
		return fmt.Errorf("failed to remove marker: %w", err)
	}
	return nil
}

func (a *AzureStorage) EnsureTrashLifecyclePolicy(retentionDays int) error {
	client, err := armstorage.NewManagementPoliciesClient(a.subscriptionID, a.armCred, nil)
	if err != nil {
		return fmt.Errorf("create Azure management policies client: %w", err)
	}

	ctx := context.Background()

	var rules []*armstorage.ManagementPolicyRule
	existing, err := client.Get(ctx, a.resourceGroup, a.accountName, armstorage.ManagementPolicyNameDefault, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if !errors.As(err, &respErr) || respErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("get existing Azure management policy: %w", err)
		}
	} else if existing.Properties != nil && existing.Properties.Policy != nil {
		rules = existing.Properties.Policy.Rules
	}

	rules = upsertAzureTrashRule(rules, retentionDays, a.containerName)

	_, err = client.CreateOrUpdate(ctx, a.resourceGroup, a.accountName, armstorage.ManagementPolicyNameDefault,
		armstorage.ManagementPolicy{
			Properties: &armstorage.ManagementPolicyProperties{
				Policy: &armstorage.ManagementPolicySchema{Rules: rules},
			},
		}, nil)
	if err != nil {
		zap.L().Error("Failed to set Azure blob lifecycle policy",
			zap.String("container", a.containerName),
			zap.Int("trashRetentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	zap.L().Info("Azure blob lifecycle policy configured",
		zap.String("container", a.containerName),
		zap.Int("trashRetentionDays", retentionDays))

	return nil
}

func upsertAzureTrashRule(
	rules []*armstorage.ManagementPolicyRule, retentionDays int, containerName string,
) []*armstorage.ManagementPolicyRule {
	trashRule := &armstorage.ManagementPolicyRule{
		Enabled: to.Ptr(true),
		Name:    to.Ptr(azureTrashLifecycleRuleName),
		Type:    to.Ptr(armstorage.RuleTypeLifecycle),
		Definition: &armstorage.ManagementPolicyDefinition{
			Filters: &armstorage.ManagementPolicyFilter{
				BlobTypes:   []*string{to.Ptr("blockBlob")},
				PrefixMatch: []*string{to.Ptr(containerName + "/" + trashPrefix)},
			},
			Actions: &armstorage.ManagementPolicyAction{
				BaseBlob: &armstorage.ManagementPolicyBaseBlob{
					Delete: &armstorage.DateAfterModification{
						DaysAfterModificationGreaterThan: to.Ptr(float32(retentionDays)),
					},
				},
			},
		},
	}

	for i, rule := range rules {
		if rule.Name != nil && *rule.Name == azureTrashLifecycleRuleName {
			rules[i] = trashRule
			return rules
		}
	}

	return append(rules, trashRule)
}
