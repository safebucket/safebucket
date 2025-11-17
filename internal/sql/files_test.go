package sql

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"api/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupMockDB creates a mock database for testing.
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock, db
}

func TestFolderExistsAtPath(t *testing.T) {
	t.Run("Root path always exists", func(t *testing.T) {
		gormDB, _, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		exists, err := FolderExistsAtPath(gormDB, bucketID, "/")

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Empty path treated as root", func(t *testing.T) {
		gormDB, _, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		exists, err := FolderExistsAtPath(gormDB, bucketID, "")

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Existing folder at root level", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		folderID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "name", "extension", "status", "bucket_id", "path", "type", "size",
			"trashed_at", "trashed_by", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			folderID, "documents", "", "", bucketID, "/", "folder", 0,
			nil, nil, now, now, nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "files" WHERE (bucket_id = $1 AND path = $2 AND name = $3 AND type = $4) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $5`,
		)).WithArgs(bucketID, "/", "documents", models.FileTypeFolder, 1).
			WillReturnRows(rows)

		exists, err := FolderExistsAtPath(gormDB, bucketID, "/documents")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Existing nested folder", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		folderID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "name", "extension", "status", "bucket_id", "path", "type", "size",
			"trashed_at", "trashed_by", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			folderID, "reports", "", "", bucketID, "/documents", "folder", 0,
			nil, nil, now, now, nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "files" WHERE (bucket_id = $1 AND path = $2 AND name = $3 AND type = $4) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $5`,
		)).WithArgs(bucketID, "/documents", "reports", models.FileTypeFolder, 1).
			WillReturnRows(rows)

		exists, err := FolderExistsAtPath(gormDB, bucketID, "/documents/reports")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Non-existent folder at root level", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "files" WHERE (bucket_id = $1 AND path = $2 AND name = $3 AND type = $4) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $5`,
		)).WithArgs(bucketID, "/", "nonexistent", models.FileTypeFolder, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		exists, err := FolderExistsAtPath(gormDB, bucketID, "/nonexistent")

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Path without leading slash", func(t *testing.T) {
		gormDB, _, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		exists, err := FolderExistsAtPath(gormDB, bucketID, "documents")

		assert.NoError(t, err)
		assert.False(t, exists, "Path without leading slash should be considered invalid")
	})
}

func TestFolderExistsAtPath_FileVsFolder(t *testing.T) {
	gormDB, mock, db := setupMockDB(t)
	defer db.Close()

	bucketID := uuid.New()

	// File query should return a FILE not FOLDER, so folder check should fail
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "files" WHERE (bucket_id = $1 AND path = $2 AND name = $3 AND type = $4) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $5`,
	)).WithArgs(bucketID, "/", "document.txt", models.FileTypeFolder, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	exists, err := FolderExistsAtPath(gormDB, bucketID, "/document.txt")

	assert.NoError(t, err)
	assert.False(t, exists, "File should not be recognized as a folder")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFileByID(t *testing.T) {
	t.Run("File exists", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		fileID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "name", "extension", "status", "bucket_id", "path", "type", "size",
			"trashed_at", "trashed_by", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			fileID, "test.txt", "txt", "uploaded", bucketID, "/", "file", 1024,
			nil, nil, now, now, nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "files" WHERE (id = $1 AND bucket_id = $2) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $3`,
		)).WithArgs(fileID, bucketID, 1).
			WillReturnRows(rows)

		result, err := GetFileByID(gormDB, bucketID, fileID)

		assert.NoError(t, err)
		assert.Equal(t, fileID, result.ID)
		assert.Equal(t, "test.txt", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("File not found", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		fileID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "files" WHERE (id = $1 AND bucket_id = $2) AND "files"."deleted_at" IS NULL ORDER BY "files"."id" LIMIT $3`,
		)).WithArgs(fileID, bucketID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		_, err := GetFileByID(gormDB, bucketID, fileID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
