package database

import (
	"database/sql"
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const fileVersionsMigrationVersion int64 = 12

func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec(pragmaForeignKeys)
	require.NoError(t, err)

	RunMigrations(sqlDB, DialectSQLite)
	RegisterCallbacks(db)

	return db
}

// migrateSQLiteUpTo stops at a chosen migration so a test can seed the schema as it stood before a
// migration and then assert what that migration did to the existing rows.
func migrateSQLiteUpTo(t *testing.T, db *sql.DB, version int64) {
	t.Helper()

	goose.SetLogger(zapGooseLogger{l: zap.L()})
	require.NoError(t, goose.SetDialect("sqlite3"))

	goose.SetBaseFS(sqliteMigrations)
	defer goose.SetBaseFS(nil)

	require.NoError(t, goose.UpTo(db, "migrations/"+DialectSQLite, version))
}

func TestSQLite_Migrations(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	tables := []string{
		"users", "buckets", "memberships", "folders", "files", "file_versions", "invites", "challenges", "mfa_devices",
	}
	for _, table := range tables {
		var count int
		err = sqlDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}

func TestSQLite_UUIDGeneration(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	t.Run("generates UUID on create when not set", func(t *testing.T) {
		user := models.User{
			Email:        "test@example.com",
			ProviderType: models.LocalProviderType,
			ProviderKey:  string(models.LocalProviderType),
			Role:         models.RoleUser,
		}

		require.Equal(t, uuid.Nil, user.ID)
		require.NoError(t, db.Create(&user).Error)
		assert.NotEqual(t, uuid.Nil, user.ID, "UUID should be generated automatically")
	})

	t.Run("preserves UUID when already set", func(t *testing.T) {
		existingID := uuid.New()
		user := models.User{
			ID:           existingID,
			Email:        "preset@example.com",
			ProviderType: models.LocalProviderType,
			ProviderKey:  string(models.LocalProviderType),
			Role:         models.RoleUser,
		}

		require.NoError(t, db.Create(&user).Error)
		assert.Equal(t, existingID, user.ID, "pre-set UUID should be preserved")
	})

	t.Run("generates unique UUIDs for batch create", func(t *testing.T) {
		user := models.User{
			Email:        "batch@example.com",
			ProviderType: models.LocalProviderType,
			ProviderKey:  string(models.LocalProviderType),
			Role:         models.RoleUser,
		}
		require.NoError(t, db.Create(&user).Error)

		bucket := models.Bucket{
			Name:      "test-bucket",
			CreatedBy: user.ID,
		}
		require.NoError(t, db.Create(&bucket).Error)

		memberships := []models.Membership{
			{UserID: user.ID, BucketID: bucket.ID, Group: models.GroupOwner},
		}

		require.NoError(t, db.Create(&memberships).Error)
		assert.NotEqual(t, uuid.Nil, memberships[0].ID)
	})
}

func TestSQLite_CRUD(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	user := models.User{
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NotEqual(t, uuid.Nil, user.ID)

	var fetched models.User
	require.NoError(t, db.First(&fetched, "id = ?", user.ID).Error)
	assert.Equal(t, "John", fetched.FirstName)
	assert.Equal(t, "john@example.com", fetched.Email)

	require.NoError(t, db.Model(&fetched).Update("first_name", "Jane").Error)
	var updated models.User
	require.NoError(t, db.First(&updated, "id = ?", user.ID).Error)
	assert.Equal(t, "Jane", updated.FirstName)

	require.NoError(t, db.Delete(&updated).Error)
	var deleted models.User
	err = db.First(&deleted, "id = ?", user.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "soft-deleted user should not be found")

	var unscoped models.User
	require.NoError(t, db.Unscoped().First(&unscoped, "id = ?", user.ID).Error)
	assert.Equal(t, user.ID, unscoped.ID)
}

func TestSQLite_ForeignKeyConstraints(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	bucket := models.Bucket{
		Name:      "orphan-bucket",
		CreatedBy: uuid.New(), // non-existent user
	}
	err = db.Create(&bucket).Error
	assert.Error(t, err, "foreign key constraint should prevent creating bucket with non-existent user")
}

func TestSQLite_FileVersionsBackfill(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	_, err = sqlDB.Exec(pragmaForeignKeys)
	require.NoError(t, err)

	migrateSQLiteUpTo(t, sqlDB, fileVersionsMigrationVersion-1)
	RegisterCallbacks(db)

	user := models.User{
		Email:        "owner@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleAdmin,
	}
	require.NoError(t, db.Create(&user).Error)

	bucket := models.Bucket{Name: "bucket", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)

	uploaded := models.File{
		Name:     "uploaded.txt",
		Status:   models.FileStatusUploaded,
		BucketID: bucket.ID,
		Size:     42,
	}
	require.NoError(t, db.Create(&uploaded).Error)

	uploading := models.File{
		Name:     "uploading.txt",
		Status:   models.FileStatusUploading,
		BucketID: bucket.ID,
	}
	require.NoError(t, db.Create(&uploading).Error)

	migrateSQLiteUpTo(t, sqlDB, fileVersionsMigrationVersion)

	existingFiles := []models.File{uploaded, uploading}

	t.Run("backfills exactly one version per existing file", func(t *testing.T) {
		var count int
		require.NoError(t, sqlDB.QueryRow("SELECT count(*) FROM file_versions").Scan(&count))
		assert.Equal(t, len(existingFiles), count)
	})

	t.Run("reuses the file id as the version id", func(t *testing.T) {
		for _, file := range existingFiles {
			var (
				id            string
				versionNumber int
				size          int
				status        string
			)
			require.NoError(t, sqlDB.QueryRow(
				"SELECT id, version_number, size, status FROM file_versions WHERE file_id = ?", file.ID,
			).Scan(&id, &versionNumber, &size, &status))

			assert.Equal(t, file.ID.String(), id)
			assert.Equal(t, 1, versionNumber)
			assert.Equal(t, file.Size, size)
			assert.Equal(t, string(file.Status), status)
		}
	})

	t.Run("points every file at its backfilled version", func(t *testing.T) {
		for _, file := range existingFiles {
			var (
				currentVersionID  string
				lastVersionNumber int
			)
			require.NoError(t, sqlDB.QueryRow(
				"SELECT current_version_id, last_version_number FROM files WHERE id = ?", file.ID,
			).Scan(&currentVersionID, &lastVersionNumber))

			assert.Equal(t, file.ID.String(), currentVersionID)
			assert.Equal(t, 1, lastVersionNumber)
		}
	})

	t.Run("rejects reusing a version number on the same file", func(t *testing.T) {
		_, insertErr := sqlDB.Exec(
			"INSERT INTO file_versions (id, file_id, version_number, size, status) VALUES (?, ?, 1, 0, 'uploaded')",
			uuid.New(), uploaded.ID,
		)
		assert.Error(t, insertErr, "unique index should prevent reissuing a version number")
	})

	t.Run("cascades version rows when the file row is deleted", func(t *testing.T) {
		_, deleteErr := sqlDB.Exec("DELETE FROM files WHERE id = ?", uploading.ID)
		require.NoError(t, deleteErr)

		var count int
		require.NoError(t, sqlDB.QueryRow(
			"SELECT count(*) FROM file_versions WHERE file_id = ?", uploading.ID,
		).Scan(&count))
		assert.Equal(t, 0, count)
	})
}
