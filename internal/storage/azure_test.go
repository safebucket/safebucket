package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
