package workers

import (
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type gcStubStorage struct {
	listUploadedPartsFn func(path, uploadID string) ([]storage.PartInfo, error)
	abortedUploadIDs    []string
}

func (s *gcStubStorage) PresignedGetObject(string, storage.GetObjectOptions) (string, error) {
	return "", nil
}

func (s *gcStubStorage) PresignUpload(string, int, map[string]string) (storage.PresignedUpload, error) {
	return storage.PresignedUpload{}, nil
}

func (s *gcStubStorage) SupportsMultipart() bool { return true }

func (s *gcStubStorage) ListUploadedParts(path, uploadID string) ([]storage.PartInfo, error) {
	if s.listUploadedPartsFn != nil {
		return s.listUploadedPartsFn(path, uploadID)
	}
	return nil, nil
}

func (s *gcStubStorage) CompleteMultipartUpload(string, string, []storage.PartInfo) error {
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

	return file
}

func countFiles(t *testing.T, db *gorm.DB, fileID uuid.UUID) int64 {
	t.Helper()

	var count int64
	require.NoError(t, db.Model(&models.File{}).Where("id = ?", fileID).Count(&count).Error)
	return count
}

func TestCleanupStaleUploads(t *testing.T) {
	staleCreatedAt := time.Now().Add(-GCStaleUploadThreshold - time.Minute)
	recentCreatedAt := time.Now()

	t.Run("non-multipart stale upload is deleted", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)

		store := &gcStubStorage{}
		worker := &GarbageCollectorWorker{DB: db, Storage: store, Cache: cache.NewMemoryCache()}

		count, err := worker.cleanupStaleUploads(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, int64(0), countFiles(t, db, file.ID))
	})

	t.Run("recent upload is left alone", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, recentCreatedAt)

		store := &gcStubStorage{}
		worker := &GarbageCollectorWorker{DB: db, Storage: store, Cache: cache.NewMemoryCache()}

		count, err := worker.cleanupStaleUploads(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
	})

	t.Run("stale multipart upload with a recent part is skipped", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)
		mem := cache.NewMemoryCache()
		require.NoError(t, cache.SetMultipartState(mem, file.ID.String(),
			cache.MultipartState{UploadID: "upload-1", PartSize: 32 * 1024 * 1024}))

		store := &gcStubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, LastModified: time.Now()}}, nil
			},
		}
		worker := &GarbageCollectorWorker{DB: db, Storage: store, Cache: mem}

		count, err := worker.cleanupStaleUploads(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
		assert.Empty(t, store.abortedUploadIDs)
	})

	t.Run("stale multipart upload with no recent activity is aborted and deleted", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)
		mem := cache.NewMemoryCache()
		require.NoError(t, cache.SetMultipartState(mem, file.ID.String(),
			cache.MultipartState{UploadID: "upload-2", PartSize: 32 * 1024 * 1024}))

		store := &gcStubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, LastModified: staleCreatedAt}}, nil
			},
		}
		worker := &GarbageCollectorWorker{DB: db, Storage: store, Cache: mem}

		count, err := worker.cleanupStaleUploads(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, int64(0), countFiles(t, db, file.ID))
		assert.Equal(t, []string{"upload-2"}, store.abortedUploadIDs)

		_, found, stateErr := cache.GetMultipartState(mem, file.ID.String())
		require.NoError(t, stateErr)
		assert.False(t, found, "multipart state should be deleted from cache after abort")
	})

	t.Run("list parts error skips the row this cycle", func(t *testing.T) {
		db := setupGCTestDB(t)
		bucket := gcTestBucket(t, db)
		file := createGCTestFile(t, db, bucket.ID, staleCreatedAt)
		mem := cache.NewMemoryCache()
		require.NoError(t, cache.SetMultipartState(mem, file.ID.String(),
			cache.MultipartState{UploadID: "upload-3", PartSize: 32 * 1024 * 1024}))

		store := &gcStubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return nil, assert.AnError
			},
		}
		worker := &GarbageCollectorWorker{DB: db, Storage: store, Cache: mem}

		count, err := worker.cleanupStaleUploads(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, int64(1), countFiles(t, db, file.ID))
		assert.Empty(t, store.abortedUploadIDs)
	})
}
