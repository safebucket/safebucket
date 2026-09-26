package fileversions

import (
	"net/http"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestCancelledFirstVersionRetainsCleanupTracking(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "cancelled.txt", Size: 4})
	_, before, err := m.Delete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, models.FileStatusUploading, before.Status)
	deleted, err := sql.GetFileVersionByID(m.DB, version.FileID, version.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStatusDeleting, deleted.Status)
	require.NotNil(t, deleted.CleanupAfter)
	assert.True(t, deleted.CleanupAfter.After(time.Now()))
	assert.Nil(t, loadFile(t, m, version.FileID).CurrentVersionID)
	key := storage.VersionObjectKey(bucket.ID, version.ID)
	store.objects[key] = map[string]string{}
	_, err = m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, true)
	assertCode(t, err, http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
	require.NoError(t, m.Cleanup(bucket.ID, version.FileID, version.ID))
	_, err = sql.GetFileVersionByID(m.DB, version.FileID, version.ID)
	require.NoError(t, err)
	store.objects[key] = map[string]string{}
	require.NoError(t, m.DB.Model(&deleted).Update("cleanup_after", time.Now().Add(-time.Minute)).Error)
	require.NoError(t, m.Cleanup(bucket.ID, version.FileID, version.ID))
	assert.Empty(t, store.objects)
	var count int64
	require.NoError(t, m.DB.Unscoped().Model(&models.File{}).Where("id = ?", version.FileID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestShareQuotaReleasedOnce(t *testing.T) {
	for _, expire := range []bool{false, true} {
		name := "cancel"
		if expire {
			name = "expire"
		}
		t.Run(name, func(t *testing.T) {
			m, store, bucket, user := setupManager(t)
			body := models.FileUploadBody{Name: "quota.txt", Size: 4}
			first := startVersion(t, m, bucket.ID, user.ID, body)
			finishVersion(t, m, store, bucket.ID, first, false)
			maxUploads := 1
			share := models.Share{BucketID: bucket.ID, CreatedBy: user.ID, Type: models.ShareTypeBucket,
				AllowUpload: true, MaxUploads: &maxUploads}
			require.NoError(t, m.DB.Create(&share).Error)
			response, err := m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
			require.NoError(t, err)
			version, err := sql.GetFileVersionByID(m.DB, first.FileID, uuid.MustParse(response.VersionID))
			require.NoError(t, err)
			if expire {
				require.NoError(t, m.DB.Model(&version).Update("created_at", time.Now().Add(-time.Hour)).Error)
			}
			for range 2 {
				if expire {
					_, err = m.ExpirePending(bucket.ID, version, time.Now().Add(-20*time.Minute))
				} else {
					err = deleteVersion(m, bucket.ID, first.FileID, version.ID, &share)
				}
				require.NoError(t, err)
			}
			require.NoError(t, m.DB.Where("id = ?", share.ID).Find(&share).Error)
			assert.Zero(t, share.CurrentUploads)
			response, err = m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
			require.NoError(t, err)
			completed, err := sql.GetFileVersionByID(m.DB, first.FileID, uuid.MustParse(response.VersionID))
			require.NoError(t, err)
			finishVersion(t, m, store, bucket.ID, completed, false)
			next := startVersion(t, m, bucket.ID, user.ID, body)
			finishVersion(t, m, store, bucket.ID, next, false)
			require.NoError(t, deleteVersion(m, bucket.ID, first.FileID, completed.ID, nil))
			require.NoError(t, m.DB.Where("id = ?", share.ID).Find(&share).Error)
			assert.Equal(t, 1, share.CurrentUploads)
			_, err = m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
			assertCode(t, err, http.StatusForbidden, apierrors.CodeShareMaxUploadsReached)
		})
	}
}

func TestLegacyConfirmationAfterRetention(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	m.MaxVersions = 1
	body := models.FileUploadBody{Name: "legacy.txt", Size: 4}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	finishVersion(t, m, store, bucket.ID, first, false)
	second := startVersion(t, m, bucket.ID, user.ID, body)
	store.objects[storage.VersionObjectKey(bucket.ID, second.ID)] = Metadata(loadFile(t, m, first.FileID), second)
	completed, err := m.Complete(zap.NewNop(), bucket.ID, first.FileID, uuid.Nil, nil, false)
	require.NoError(t, err)
	assert.True(t, completed.Changed)
	assert.Equal(t, second.ID, completed.Version.ID)
	require.NoError(t, m.Cleanup(bucket.ID, first.FileID, first.ID))
	completed, err = m.Complete(zap.NewNop(), bucket.ID, first.FileID, uuid.Nil, nil, false)
	require.NoError(t, err)
	assert.False(t, completed.Changed)
	assert.Equal(t, &second.ID, loadFile(t, m, first.FileID).CurrentVersionID)
	fresh := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "empty.txt", Size: 4})
	require.NoError(t, deleteVersion(m, bucket.ID, fresh.FileID, fresh.ID, nil))
	_, err = m.Complete(zap.NewNop(), bucket.ID, fresh.FileID, uuid.Nil, nil, false)
	assertCode(t, err, http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
}

func TestBatchDeletionRetainsRecordsOnPartialFailure(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	var files []models.File
	for _, name := range []string{"first.txt", "second.txt"} {
		version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: name, Size: 4})
		finishVersion(t, m, store, bucket.ID, version, false)
		files = append(files, loadFile(t, m, version.FileID))
	}
	batch := &batchFailureStorage{memoryStorage: store, fail: true}
	m.Storage = batch
	remove := func(tx *gorm.DB) error {
		if err := m.RemoveFileObjects(tx, files...); err != nil {
			return err
		}
		return tx.Unscoped().Delete(&files).Error
	}
	require.Error(t, m.DB.Transaction(remove))
	var count int64
	require.NoError(t, m.DB.Model(&models.FileVersion{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
	assert.Len(t, store.objects, 1)
	batch.fail = false
	require.NoError(t, m.DB.Transaction(remove))
	assert.Empty(t, store.objects)
	require.NoError(t, m.DB.Model(&models.FileVersion{}).Count(&count).Error)
	assert.Zero(t, count)
	assert.Equal(t, 2, batch.calls)
}

func TestMultipartCompletionFailureCode(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	store.multipart = true
	version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "large.txt", Size: 6})
	store.parts = []storage.PartInfo{{PartNumber: 1, Size: 4}, {PartNumber: 2, Size: 2}}
	m.Storage = multipartFailureStorage{store}
	_, err := m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, false)
	assertCode(t, err, http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
}

type batchFailureStorage struct {
	*memoryStorage

	fail  bool
	calls int
}

func (s *batchFailureStorage) RemoveObjects(keys []string) error {
	s.calls++
	for _, key := range keys {
		delete(s.objects, key)
		if s.fail {
			return assert.AnError
		}
	}
	return nil
}

type multipartFailureStorage struct{ *memoryStorage }

func (multipartFailureStorage) CompleteMultipartUpload(string, string, []storage.PartInfo, map[string]string) error {
	return assert.AnError
}
