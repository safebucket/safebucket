package database

import (
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSQLiteDB creates an in-memory SQLite database with migrations and callbacks.
func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec(pragmaForeignKeys)
	require.NoError(t, err)

	runMigrations(sqlDB, "sqlite")
	RegisterCallbacks(db)

	return db
}

func TestSQLite_Migrations(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// Verify all expected tables exist
	tables := []string{"users", "buckets", "memberships", "folders", "files", "invites", "challenges", "mfa_devices"}
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
		// First create a user and bucket for foreign key constraints
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

	// Create
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

	// Read
	var fetched models.User
	require.NoError(t, db.First(&fetched, "id = ?", user.ID).Error)
	assert.Equal(t, "John", fetched.FirstName)
	assert.Equal(t, "john@example.com", fetched.Email)

	// Update
	require.NoError(t, db.Model(&fetched).Update("first_name", "Jane").Error)
	var updated models.User
	require.NoError(t, db.First(&updated, "id = ?", user.ID).Error)
	assert.Equal(t, "Jane", updated.FirstName)

	// Soft delete
	require.NoError(t, db.Delete(&updated).Error)
	var deleted models.User
	err = db.First(&deleted, "id = ?", user.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "soft-deleted user should not be found")

	// Unscoped should still find it
	var unscoped models.User
	require.NoError(t, db.Unscoped().First(&unscoped, "id = ?", user.ID).Error)
	assert.Equal(t, user.ID, unscoped.ID)
}

func TestSQLite_ForeignKeyConstraints(t *testing.T) {
	db := setupSQLiteDB(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	// Try to create a bucket referencing a non-existent user
	bucket := models.Bucket{
		Name:      "orphan-bucket",
		CreatedBy: uuid.New(), // non-existent user
	}
	err = db.Create(&bucket).Error
	assert.Error(t, err, "foreign key constraint should prevent creating bucket with non-existent user")
}
