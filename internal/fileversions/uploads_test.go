package fileversions

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/database"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUploadVersions(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	expires := time.Now().Add(time.Hour)
	body := models.FileUploadBody{Name: "report.txt", Size: 5, ExpiresAt: &expires}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	assert.Equal(t, first.ID, first.FileID)
	assert.Equal(t, 1, first.Version)
	assert.Equal(t, &user.ID, first.UploadedBy)
	assert.Nil(t, loadFile(t, m, first.FileID).CurrentVersionID)
	_, err := m.Start(zap.NewNop(), bucket.ID, body, &user.ID, nil)
	assertCode(t, err, http.StatusConflict, apierrors.CodeFileUploadInProgress)
	_, err = m.Complete(zap.NewNop(), bucket.ID, first.FileID, first.ID, nil, false)
	assertCode(t, err, http.StatusNotFound, apierrors.CodeFileNotInStorage)
	finishVersion(t, m, store, bucket.ID, first, true)
	second := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: body.Name, Size: 9})
	assert.Equal(t, first.FileID, second.FileID)
	assert.Equal(t, 2, second.Version)
	assert.NotEqual(t, first.ID, second.ID)
	file := loadFile(t, m, first.FileID)
	assert.Equal(t, &first.ID, file.CurrentVersionID)
	assert.Equal(t, 5, file.Size)
	assert.WithinDuration(t, expires, *file.ExpiresAt, time.Second)
	assert.Equal(t, models.FileStatusUploaded, file.Status)
	finishVersion(t, m, store, bucket.ID, second, false)
	file = loadFile(t, m, first.FileID)
	assert.Equal(t, &second.ID, file.CurrentVersionID)
	assert.Equal(t, 9, file.Size)
	assert.Contains(t, store.objects, storage.VersionObjectKey(bucket.ID, first.ID))
	duplicate, err := m.Complete(zap.NewNop(), bucket.ID, file.ID, second.ID, nil, true)
	require.NoError(t, err)
	assert.False(t, duplicate.Changed)
	legacy, err := m.Complete(zap.NewNop(), bucket.ID, file.ID, file.ID, nil, false)
	require.NoError(t, err)
	assert.False(t, legacy.Changed)
	assert.Equal(t, &second.ID, loadFile(t, m, file.ID).CurrentVersionID)

	folder := models.Folder{Name: "other", BucketID: bucket.ID, Status: models.FolderStatusCreated}
	require.NoError(t, m.DB.Create(&folder).Error)
	body.FolderID = &folder.ID
	other := startVersion(t, m, bucket.ID, user.ID, body)
	assert.NotEqual(t, first.FileID, other.FileID)
	assert.Equal(t, 1, other.Version)
}

func TestVersionNumberReuseAfterCleanup(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	body := models.FileUploadBody{Name: "reuse.txt", Size: 4}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	finishVersion(t, m, store, bucket.ID, first, false)
	cancelled := startVersion(t, m, bucket.ID, user.ID, body)
	require.NoError(t, deleteVersion(m, bucket.ID, first.FileID, cancelled.ID, nil))
	require.NoError(t, m.DB.Model(&cancelled).
		Update("cleanup_after", time.Now().Add(-time.Minute)).Error)
	require.NoError(t, m.Cleanup(bucket.ID, first.FileID, cancelled.ID))

	replacement := startVersion(t, m, bucket.ID, user.ID, body)
	assert.Equal(t, first.FileID, replacement.FileID)
	assert.Equal(t, 2, replacement.Version)
	assert.NotEqual(t, cancelled.ID, replacement.ID)
	finishVersion(t, m, store, bucket.ID, replacement, false)
	assert.Equal(t, &replacement.ID, loadFile(t, m, first.FileID).CurrentVersionID)
}

func TestVersionCancellationAndRetention(t *testing.T) {
	for _, limit := range []int{1, 5} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			m, store, bucket, user := setupManager(t)
			m.MaxVersions = limit
			body := models.FileUploadBody{Name: "history.txt", Size: 4}
			first := startVersion(t, m, bucket.ID, user.ID, body)
			finishVersion(t, m, store, bucket.ID, first, false)
			cancelled := startVersion(t, m, bucket.ID, user.ID, body)
			require.NoError(t, deleteVersion(m, bucket.ID, first.FileID, cancelled.ID, nil))
			store.objects[storage.VersionObjectKey(bucket.ID, cancelled.ID)] = Metadata(
				loadFile(t, m, first.FileID),
				cancelled,
			)
			_, err := m.Complete(zap.NewNop(), bucket.ID, first.FileID, cancelled.ID, nil, true)
			assertCode(t, err, http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
			assert.Equal(t, &first.ID, loadFile(t, m, first.FileID).CurrentVersionID)
			var current models.FileVersion
			for i := range 6 {
				current = startVersion(t, m, bucket.ID, user.ID, body)
				assert.Equal(t, i+3, current.Version)
				finishVersion(t, m, store, bucket.ID, current, false)
			}
			var retained int64
			require.NoError(
				t,
				m.DB.Model(&models.FileVersion{}).
					Where("file_id = ? AND status = ?", first.FileID, models.FileStatusUploaded).
					Count(&retained).
					Error,
			)
			assert.Equal(t, int64(limit), retained)
			assertCode(
				t,
				deleteVersion(m, bucket.ID, first.FileID, current.ID, nil),
				http.StatusConflict,
				apierrors.CodeFileVersionIsCurrent,
			)
			store.removeError = assert.AnError
			require.Error(t, m.Cleanup(bucket.ID, first.FileID, first.ID))
			require.NoError(t, m.DB.Model(&models.FileVersion{}).Where("id = ?", first.ID).Count(&retained).Error)
			assert.Equal(t, int64(1), retained)
			store.removeError = nil
			past := time.Now().Add(-time.Minute)
			require.NoError(
				t,
				m.DB.Model(&models.FileVersion{}).Where("id = ?", cancelled.ID).Update("cleanup_after", past).Error,
			)
			var deleted []models.FileVersion
			require.NoError(
				t,
				m.DB.Where("file_id = ? AND status = ?", first.FileID, models.FileStatusDeleting).Find(&deleted).Error,
			)
			for _, version := range deleted {
				require.NoError(t, m.Cleanup(bucket.ID, first.FileID, version.ID))
			}
			assert.Len(t, store.objects, limit)
			file := loadFile(t, m, first.FileID)
			require.NoError(t, m.DB.Transaction(func(tx *gorm.DB) error {
				if removeErr := m.RemoveFileObjects(tx, file); removeErr != nil {
					return removeErr
				}
				return tx.Unscoped().Delete(&models.File{}, first.FileID).Error
			}))
			assert.Empty(t, store.objects)
		})
	}
}

func TestShareUploadVersions(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	body := models.FileUploadBody{Name: "public.txt", Size: 4}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	finishVersion(t, m, store, bucket.ID, first, false)
	maxUploads := 1
	share := models.Share{
		BucketID:    bucket.ID,
		CreatedBy:   user.ID,
		Type:        models.ShareTypeBucket,
		AllowUpload: true,
		MaxUploads:  &maxUploads,
	}
	require.NoError(t, m.DB.Create(&share).Error)
	response, err := m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
	require.NoError(t, err)
	version, err := sql.GetFileVersionByID(m.DB, first.FileID, uuid.MustParse(response.VersionID))
	require.NoError(t, err)
	assert.Nil(t, version.UploadedBy)
	assert.Equal(t, &share.ID, version.ShareID)
	other := models.Share{BucketID: bucket.ID, CreatedBy: user.ID, Type: models.ShareTypeBucket, AllowUpload: true}
	require.NoError(t, m.DB.Create(&other).Error)
	assertCode(
		t,
		deleteVersion(m, bucket.ID, first.FileID, version.ID, &other),
		http.StatusForbidden,
		apierrors.CodeForbidden,
	)
	store.objects[storage.VersionObjectKey(bucket.ID, version.ID)] = Metadata(loadFile(t, m, first.FileID), version)
	completed, err := m.Complete(zap.NewNop(), bucket.ID, first.FileID, version.ID, &share, false)
	require.NoError(t, err)
	assert.True(t, completed.Changed)
	assertCode(
		t,
		deleteVersion(m, bucket.ID, first.FileID, version.ID, &share),
		http.StatusConflict,
		apierrors.CodeFileVersionNotDeletable,
	)
	_, err = m.Start(zap.NewNop(), bucket.ID, body, nil, &share)
	assertCode(t, err, http.StatusForbidden, apierrors.CodeShareMaxUploadsReached)
	other.AllowUpload = false
	require.NoError(t, m.DB.Save(&other).Error)
	_, err = m.Complete(zap.NewNop(), bucket.ID, first.FileID, version.ID, &other, false)
	assertCode(t, err, http.StatusForbidden, apierrors.CodeShareUploadNotAllowed)
}

func TestMultipartVersionCompletion(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	store.multipart = true
	version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "large.txt", Size: 6})
	state, found, err := cache.GetMultipartState(m.Cache, version.ID.String())
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(4), state.PartSize)
	completed, err := m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, true)
	require.NoError(t, err)
	assert.False(t, completed.Changed)
	store.parts = []storage.PartInfo{{PartNumber: 1, Size: 4}}
	_, err = m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, false)
	assertCode(t, err, http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	store.parts = append(store.parts, storage.PartInfo{PartNumber: 2, Size: 2})
	completed, err = m.Complete(zap.NewNop(), bucket.ID, version.FileID, version.ID, nil, false)
	require.NoError(t, err)
	assert.True(t, completed.Changed)
	assert.Equal(t, 1, store.completions)
	assert.Equal(t, version.ID.String(), store.objects[storage.VersionObjectKey(bucket.ID, version.ID)]["version_id"])
	_, found, err = cache.GetMultipartState(m.Cache, version.ID.String())
	require.NoError(t, err)
	assert.False(t, found)
}

func TestMultipartVersionCancellation(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	store.multipart = true
	version := startVersion(t, m, bucket.ID, user.ID, models.FileUploadBody{Name: "large.txt", Size: 6})
	state, found, err := cache.GetMultipartState(m.Cache, version.ID.String())
	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, deleteVersion(m, bucket.ID, version.FileID, version.ID, nil))
	assert.Equal(t, []string{state.UploadID}, store.aborted)
	_, found, err = cache.GetMultipartState(m.Cache, version.ID.String())
	require.NoError(t, err)
	assert.False(t, found)
}

func TestPendingVersionCannotResurrectTrashedFile(t *testing.T) {
	m, store, bucket, user := setupManager(t)
	body := models.FileUploadBody{Name: "trash.txt", Size: 4}
	first := startVersion(t, m, bucket.ID, user.ID, body)
	finishVersion(t, m, store, bucket.ID, first, false)
	pending := startVersion(t, m, bucket.ID, user.ID, body)
	require.NoError(t, m.DB.Delete(&models.File{}, first.FileID).Error)
	store.objects[storage.VersionObjectKey(bucket.ID, pending.ID)] = map[string]string{}
	_, err := m.Complete(zap.NewNop(), bucket.ID, first.FileID, pending.ID, nil, true)
	assertCode(t, err, http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
}

type memoryStorage struct {
	storage.IStorage

	objects     map[string]map[string]string
	removeError error
	multipart   bool
	parts       []storage.PartInfo
	completions int
	aborted     []string
}

func (s *memoryStorage) PresignUpload(key string, _ int, _ map[string]string) (storage.PresignedUpload, error) {
	response := storage.PresignedUpload{Response: models.FileUploadResponse{Method: "put", URL: key}}
	if s.multipart {
		response.UploadID = uuid.NewString()
		response.PartSize = 4
	}
	return response, nil
}

func (s *memoryStorage) StatObject(key string) (map[string]string, error) {
	if value, ok := s.objects[key]; ok {
		return value, nil
	}
	return nil, errors.New("object missing")
}

func (s *memoryStorage) RemoveObject(key string) error {
	if s.removeError != nil {
		return s.removeError
	}
	delete(s.objects, key)
	return nil
}

func (s *memoryStorage) RemoveObjects(keys []string) error {
	for _, key := range keys {
		if err := s.RemoveObject(key); err != nil {
			return err
		}
	}
	return nil
}

func (s *memoryStorage) AbortMultipartUpload(_, uploadID string) error {
	s.aborted = append(s.aborted, uploadID)
	return nil
}

func (s *memoryStorage) ListObjectParts(string, string) ([]storage.PartInfo, error) {
	return s.parts, nil
}

func (s *memoryStorage) CompleteMultipartUpload(key, _ string, _ []storage.PartInfo, metadata map[string]string) error {
	s.objects[key] = metadata
	s.completions++
	return nil
}

func setupManager(t *testing.T) (Manager, *memoryStorage, models.Bucket, models.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_fk=on"), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	database.RunMigrations(connection, database.DialectSQLite)
	database.RegisterCallbacks(db)
	user := models.User{
		Email:        "versions@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  "local",
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)
	bucket := models.Bucket{Name: "versions", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)
	store := &memoryStorage{objects: make(map[string]map[string]string)}
	memoryCache := cache.NewMemoryCache()
	t.Cleanup(memoryCache.Close)
	return Manager{DB: db, Storage: store, Cache: memoryCache, MaxVersions: 5}, store, bucket, user
}

func startVersion(t *testing.T, m Manager, bucketID, userID uuid.UUID, body models.FileUploadBody) models.FileVersion {
	t.Helper()
	response, err := m.Start(zap.NewNop(), bucketID, body, &userID, nil)
	require.NoError(t, err)
	version, err := sql.GetFileVersionByID(m.DB, uuid.MustParse(response.ID), uuid.MustParse(response.VersionID))
	require.NoError(t, err)
	return version
}

func deleteVersion(m Manager, bucketID, fileID, versionID uuid.UUID, share *models.Share) error {
	_, _, err := m.Delete(zap.NewNop(), bucketID, fileID, versionID, share)
	return err
}

func loadFile(t *testing.T, m Manager, fileID uuid.UUID) models.File {
	t.Helper()
	var file models.File
	require.NoError(t, m.DB.Where("id = ?", fileID).Find(&file).Error)
	return file
}

func finishVersion(
	t *testing.T,
	m Manager,
	store *memoryStorage,
	bucketID uuid.UUID,
	version models.FileVersion,
	notification bool,
) {
	t.Helper()
	store.objects[storage.VersionObjectKey(bucketID, version.ID)] = Metadata(loadFile(t, m, version.FileID), version)
	completed, err := m.Complete(zap.NewNop(), bucketID, version.FileID, version.ID, nil, notification)
	require.NoError(t, err)
	assert.True(t, completed.Changed)
}

func assertCode(t *testing.T, err error, status int, code string) {
	t.Helper()
	var apiErr *apierrors.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, status, apiErr.Status)
	assert.Equal(t, code, apiErr.Code)
}
