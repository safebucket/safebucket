package events

import (
	"strings"
	"testing"

	"github.com/safebucket/safebucket/internal/activity"
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

func TestVersionDeletionDoesNotExpireTrashedFile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:?_fk=on"), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	database.RunMigrations(connection, database.DialectSQLite)
	database.RegisterCallbacks(db)
	user := models.User{
		Email:        "trash@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  "local",
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)
	bucket := models.Bucket{Name: "trash", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)
	file := models.File{BucketID: bucket.ID, Name: "report.txt", Status: models.FileStatusDeleted, Size: 5}
	require.NoError(t, db.Create(&file).Error)
	first := models.FileVersion{ID: file.ID, FileID: file.ID, Version: 1, Size: 4, Status: models.FileStatusDeleting}
	current := models.FileVersion{FileID: file.ID, Version: 2, Size: 5, Status: models.FileStatusUploaded}
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&current).Error)
	require.NoError(t, db.Model(&file).Update("current_version_id", current.ID).Error)
	require.NoError(t, db.Delete(&file).Error)
	store := &expirationStorage{bucketID: bucket.ID}
	memoryCache := cache.NewMemoryCache()
	t.Cleanup(memoryCache.Close)
	params := &EventParams{
		DB:             db,
		Storage:        store,
		ActivityLogger: expirationActivity{},
		Versions:       fileversions.Manager{DB: db, Storage: store, Cache: memoryCache},
	}
	event := NewTrashExpirationFromBucketEvent(bucket.ID, "buckets/"+bucket.ID.String()+"/"+file.ID.String())
	require.NoError(t, event.callback(params))
	var count int64
	require.NoError(t, db.Unscoped().Model(&models.File{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assert.Empty(t, store.removed)
	event = NewTrashExpirationFromBucketEvent(bucket.ID, "trash/"+bucket.ID.String()+"/files/"+file.ID.String())
	require.NoError(t, event.callback(params))
	require.NoError(t, db.Unscoped().Model(&models.File{}).Count(&count).Error)
	assert.Zero(t, count)
	assert.ElementsMatch(t, []string{
		"buckets/" + bucket.ID.String() + "/" + first.ID.String(),
		"buckets/" + bucket.ID.String() + "/" + current.ID.String(),
	}, store.removed)
}

type expirationStorage struct {
	storage.IStorage

	bucketID uuid.UUID
	removed  []string
}

func (s *expirationStorage) RemoveObjects(keys []string) error {
	s.removed = append(s.removed, keys...)
	return nil
}

func (s *expirationStorage) IsTrashMarkerPath(key string) (bool, string) {
	prefix := "trash/" + s.bucketID.String() + "/files/"
	if !strings.HasPrefix(key, prefix) {
		return false, ""
	}
	return true, "buckets/" + s.bucketID.String() + "/" + strings.TrimPrefix(key, prefix)
}

type expirationActivity struct{ activity.IActivityLogger }

func (expirationActivity) Send(models.Activity) error { return nil }
