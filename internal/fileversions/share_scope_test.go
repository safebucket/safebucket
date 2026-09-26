package fileversions

import (
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFolderShareUploadScope(t *testing.T) {
	m, _, bucket, user := setupManager(t)
	root := models.Folder{Name: "shared", BucketID: bucket.ID, Status: models.FolderStatusCreated}
	require.NoError(t, m.DB.Create(&root).Error)
	child := models.Folder{Name: "nested", BucketID: bucket.ID, FolderID: &root.ID, Status: models.FolderStatusCreated}
	require.NoError(t, m.DB.Create(&child).Error)
	share := models.Share{
		BucketID:    bucket.ID,
		CreatedBy:   user.ID,
		Type:        models.ShareTypeFolder,
		FolderID:    &root.ID,
		AllowUpload: true,
	}
	require.NoError(t, m.DB.Create(&share).Error)
	for _, folder := range []*uuid.UUID{&root.ID, &child.ID} {
		response, err := m.Start(
			zap.NewNop(),
			bucket.ID,
			models.FileUploadBody{Name: "report.txt", Size: 5, FolderID: folder},
			nil,
			&share,
		)
		require.NoError(t, err)
		file := loadFile(t, m, uuid.MustParse(response.ID))
		assert.Equal(t, folder, file.FolderID)
	}
	_, err := m.Start(zap.NewNop(), bucket.ID, models.FileUploadBody{Name: "outside.txt", Size: 5}, nil, &share)
	assertCode(t, err, http.StatusForbidden, apierrors.CodeShareFileNotInShare)
	var count int64
	require.NoError(t, m.DB.Model(&models.File{}).Where("name = ?", "outside.txt").Count(&count).Error)
	assert.Zero(t, count)
}

func TestConcurrentVersionConfirmation(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "race.txt", Size: 5})
	store.objects["buckets/"+bucket.ID.String()+"/"+version.ID.String()] = map[string]string{}
	type result struct {
		completion Completion
		err        error
	}
	results := make(chan result, 2)
	for _, notification := range []bool{false, true} {
		go func() {
			completion, err := m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, notification)
			results <- result{completion, err}
		}()
	}
	changes := 0
	for range 2 {
		outcome := <-results
		require.NoError(t, outcome.err)
		if outcome.completion.Changed {
			changes++
		}
	}
	assert.Equal(t, 1, changes)
}

func TestConcurrentShareConfirmationAndCancellation(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	share := models.Share{BucketID: bucket.ID, CreatedBy: user.ID, Type: models.ShareTypeBucket, AllowUpload: true}
	require.NoError(t, m.DB.Create(&share).Error)
	body := models.FileUploadBody{Name: "race.txt", Size: 4}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	finishVersion(t, m, store, bucket.ID, first, false)
	response, err := m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
	require.NoError(t, err)
	versionID := uuid.MustParse(response.VersionID)
	store.objects[storage.VersionObjectKey(bucket.ID, versionID)] = map[string]string{}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, completeErr := m.Complete(zap.NewNop(), bucket.ID, first.FileID, versionID, &share, false)
		results <- completeErr
	}()
	go func() {
		<-start
		results <- deleteVersion(m, bucket.ID, first.FileID, versionID, &share)
	}()
	close(start)
	successes := 0
	for range 2 {
		if result := <-results; result == nil {
			successes++
		} else {
			var apiErr *apierrors.APIError
			require.ErrorAs(t, result, &apiErr)
			assert.Equal(t, http.StatusConflict, apiErr.Status)
		}
	}
	assert.Equal(t, 1, successes)
	require.NoError(t, m.DB.Where("id = ?", share.ID).Find(&share).Error)
	version, err := sql.GetFileVersionByID(m.DB, first.FileID, versionID)
	require.NoError(t, err)
	if version.Status == models.FileStatusUploaded {
		assert.Equal(t, 1, share.CurrentUploads)
	} else {
		assert.Equal(t, models.FileStatusDeleting, version.Status)
		assert.Zero(t, share.CurrentUploads)
	}
}
