package storage

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertAzureTrashRulePrefixIncludesContainer(t *testing.T) {
	t.Run("appends a container-qualified rule", func(t *testing.T) {
		rules := upsertAzureTrashRule(nil, 30, "safebucket")

		require.Len(t, rules, 1)
		filter := rules[0].Definition.Filters
		require.Len(t, filter.PrefixMatch, 1)
		assert.Equal(t, "safebucket/trash/", *filter.PrefixMatch[0])
		assert.Equal(t, azureTrashLifecycleRuleName, *rules[0].Name)
		require.Len(t, filter.BlobTypes, 1)
		assert.Equal(t, "blockBlob", *filter.BlobTypes[0])
	})

	t.Run("replaces the existing trash rule in place and keeps others", func(t *testing.T) {
		existing := []*armstorage.ManagementPolicyRule{
			{Name: to.Ptr("other-rule")},
			{Name: to.Ptr(azureTrashLifecycleRuleName)},
		}

		rules := upsertAzureTrashRule(existing, 15, "my-container")

		require.Len(t, rules, 2)
		assert.Equal(t, "other-rule", *rules[0].Name)
		assert.Equal(t, "my-container/trash/", *rules[1].Definition.Filters.PrefixMatch[0])
	})
}

func TestAzureCommitMetadata(t *testing.T) {
	objectPath := bucketsPrefix + "bucket-uuid/file-uuid"

	t.Run("carries user_id alongside path-derived ids", func(t *testing.T) {
		meta := azureCommitMetadata(objectPath, map[string]string{
			"bucket_id": "bucket-uuid",
			"file_id":   "file-uuid",
			"user_id":   "user-uuid",
		})

		require.NotNil(t, meta["user_id"])
		assert.Equal(t, "user-uuid", *meta["user_id"])
		assert.Equal(t, "bucket-uuid", *meta["bucket_id"])
		assert.Equal(t, "file-uuid", *meta["file_id"])
		assert.NotContains(t, meta, "share_id")
	})

	t.Run("carries share_id for share uploads", func(t *testing.T) {
		meta := azureCommitMetadata(objectPath, map[string]string{"share_id": "share-uuid"})

		require.NotNil(t, meta["share_id"])
		assert.Equal(t, "share-uuid", *meta["share_id"])
	})

	t.Run("falls back to path-derived ids when metadata is nil", func(t *testing.T) {
		meta := azureCommitMetadata(objectPath, nil)

		assert.Equal(t, "bucket-uuid", *meta["bucket_id"])
		assert.Equal(t, "file-uuid", *meta["file_id"])
		assert.NotContains(t, meta, "user_id")
	})

	t.Run("drops empty values", func(t *testing.T) {
		meta := azureCommitMetadata(objectPath, map[string]string{"user_id": ""})

		assert.NotContains(t, meta, "user_id")
	})
}
