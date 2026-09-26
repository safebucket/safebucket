package workers

import (
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type gcStubStorage struct {
	listObjectPartsFn func(path, uploadID string) ([]storage.PartInfo, error)
	abortedUploadIDs  []string
}

func (s *gcStubStorage) PresignedGetObject(string, storage.GetObjectOptions) (string, error) {
	return "", nil
}

func (s *gcStubStorage) PresignUpload(string, int, map[string]string) (storage.PresignedUpload, error) {
	return storage.PresignedUpload{}, nil
}

func (s *gcStubStorage) SupportsMultipart() bool { return true }

func (s *gcStubStorage) ListObjectParts(path, uploadID string) ([]storage.PartInfo, error) {
	if s.listObjectPartsFn != nil {
		return s.listObjectPartsFn(path, uploadID)
	}
	return nil, nil
}

func (s *gcStubStorage) CompleteMultipartUpload(string, string, []storage.PartInfo, map[string]string) error {
	return nil
}

func (s *gcStubStorage) AbortMultipartUpload(_, uploadID string) error {
	s.abortedUploadIDs = append(s.abortedUploadIDs, uploadID)
	return nil
}

func (s *gcStubStorage) StatObject(string) (map[string]string, error) { return nil, nil }
func (s *gcStubStorage) ListObjects(string, int32) ([]string, error)  { return nil, nil }
func (s *gcStubStorage) RemoveObject(string) error                    { return nil }
func (s *gcStubStorage) RemoveObjects([]string) error                 { return nil }
func (s *gcStubStorage) EnsureTrashLifecyclePolicy(int) error         { return nil }
func (s *gcStubStorage) MarkAsTrashed(string, any) error              { return nil }
func (s *gcStubStorage) UnmarkAsTrashed(string, any) error            { return nil }
func (s *gcStubStorage) IsTrashMarkerPath(string) (bool, string)      { return false, "" }
func (s *gcStubStorage) GetBucketName() string                        { return "" }

func setupGCTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	database.RunMigrations(sqlDB, database.DialectSQLite)
	database.RegisterCallbacks(db)

	return db
}

func gcTestBucket(t *testing.T, db *gorm.DB) models.Bucket {
	t.Helper()

	user := models.User{
		Email:        "gc-test-" + uuid.NewString() + "@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)

	bucket := models.Bucket{Name: "gc-test-bucket-" + uuid.NewString(), CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)

	return bucket
}

// UpdateColumn bypasses GORM's auto-timestamp hook so the backdated created_at sticks.
func createGCTestFile(t *testing.T, db *gorm.DB, bucketID uuid.UUID, createdAt time.Time) models.File {
	t.Helper()

	file := models.File{
		Name:     "file-" + uuid.NewString(),
		Status:   models.FileStatusUploading,
		BucketID: bucketID,
		Size:     1024,
	}
	require.NoError(t, db.Create(&file).Error)
	require.NoError(t, db.Model(&file).UpdateColumn("created_at", createdAt).Error)
	require.NoError(t, db.Create(&models.FileVersion{ID: file.ID, FileID: file.ID, Version: 1,
		Size: file.Size, Status: models.FileStatusUploading, CreatedAt: createdAt}).Error)

	return file
}

func countFiles(t *testing.T, db *gorm.DB, fileID uuid.UUID) int64 {
	t.Helper()

	var count int64
	require.NoError(t, db.Model(&models.File{}).Where("id = ?", fileID).Count(&count).Error)
	return count
}

func newGCWorker(db *gorm.DB, store storage.IStorage) *GarbageCollectorWorker {
	memoryCache := cache.NewMemoryCache()
	return &GarbageCollectorWorker{
		DB:       db,
		Storage:  store,
		Cache:    memoryCache,
		Versions: fileversions.Manager{DB: db, Storage: store, Cache: memoryCache},
	}
}

func expireStaleUploads(t *testing.T, worker *GarbageCollectorWorker) int {
	t.Helper()
	count, err := worker.cleanupStaleUploads(t.Context())
	require.NoError(t, err)
	_, err = worker.cleanupDeletedVersions(t.Context())
	require.NoError(t, err)
	return count
}

func setUploadID(t *testing.T, worker *GarbageCollectorWorker, versionID uuid.UUID, uploadID string) {
	t.Helper()
	require.NoError(t, cache.SetMultipartState(worker.Cache, versionID.String(), cache.MultipartState{
		UploadID: uploadID, PartSize: 32 * 1024 * 1024,
	}))
}

func TestCleanupStaleReplacement(t *testing.T) {
	db := setupGCTestDB(t)
	bucket := gcTestBucket(t, db)
	file := createGCTestFile(t, db, bucket.ID, time.Now())
	require.NoError(
		t,
		db.Model(&models.FileVersion{}).Where("id = ?", file.ID).Update("status", models.FileStatusUploaded).Error,
	)
	require.NoError(t, db.Model(&file).Updates(map[string]any{
		"status": models.FileStatusUploaded, "current_version_id": file.ID,
	}).Error)
	stale := models.FileVersion{FileID: file.ID, Version: 2, Size: 99, Status: models.FileStatusUploading,
		CreatedAt: time.Now().Add(-GCStaleUploadThreshold - time.Minute)}
	require.NoError(t, db.Create(&stale).Error)
	assert.Equal(t, 1, expireStaleUploads(t, newGCWorker(db, &gcStubStorage{})))
	assert.Equal(t, int64(1), countFiles(t, db, file.ID))
	var remaining models.File
	require.NoError(t, db.Where("id = ?", file.ID).Find(&remaining).Error)
	assert.Equal(t, &file.ID, remaining.CurrentVersionID)
	assert.Equal(t, file.Size, remaining.Size)
	var versions int64
	require.NoError(t, db.Model(&models.FileVersion{}).Where("file_id = ?", file.ID).Count(&versions).Error)
	assert.Equal(t, int64(1), versions)
}

func TestCleanupStaleUploads(t *testing.T) {
	staleCreatedAt := time.Now().Add(-GCStaleUploadThreshold - time.Minute)
	recentCreatedAt := time.Now()

	t.Run("non-multipart stale upload is deleted", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)

		store := &gcStubStorage{}
		assert.Equal(t, 1, expireStaleUploads(t, newGCWorker(db, store)))
		assert.Equal(t, int64(0), countFiles(t, db, file.ID))
	})

	t.Run("recent upload is left alone", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, recentCreatedAt)

		store := &gcStubStorage{}
		assert.Equal(t, 0, expireStaleUploads(t, newGCWorker(db, store)))
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
	})

	t.Run("stale multipart upload with a recent part is skipped", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)

		store := &gcStubStorage{
			listObjectPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, LastModified: time.Now()}}, nil
			},
		}
		worker := newGCWorker(db, store)
		setUploadID(t, worker, file.ID, "upload-1")
		assert.Equal(t, 0, expireStaleUploads(t, worker))
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
		assert.Empty(t, store.abortedUploadIDs)
	})

	t.Run("stale multipart upload with no recent activity is aborted and deleted", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)

		store := &gcStubStorage{
			listObjectPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, LastModified: staleCreatedAt}}, nil
			},
		}
		worker := newGCWorker(db, store)
		setUploadID(t, worker, file.ID, "upload-2")
		assert.Equal(t, 1, expireStaleUploads(t, worker))
		assert.Equal(t, int64(0), countFiles(t, db, file.ID))
		assert.Equal(t, []string{"upload-2"}, store.abortedUploadIDs)
	})

	t.Run("list parts error skips the row this cycle", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)

		store := &gcStubStorage{
			listObjectPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return nil, assert.AnError
			},
		}
		worker := newGCWorker(db, store)
		setUploadID(t, worker, file.ID, "upload-3")
		assert.Equal(t, 0, expireStaleUploads(t, worker))
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
		assert.Empty(t, store.abortedUploadIDs)
	})
}
