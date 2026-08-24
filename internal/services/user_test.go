package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	database.RunMigrations(sqlDB, database.DialectSQLite)
	database.RegisterCallbacks(db)

	return db
}

func TestGetUserStats(t *testing.T) {
	t.Run("counts only files and buckets with active membership", func(t *testing.T) {
		db := setupUserServiceTestDB(t)
		service := UserService{DB: db}

		user := models.User{
			Email:        "stats-user@example.com",
			ProviderType: models.LocalProviderType,
			ProviderKey:  string(models.LocalProviderType),
			Role:         models.RoleUser,
		}
		require.NoError(t, db.Create(&user).Error)

		activeBucket := models.Bucket{Name: "active-bucket", CreatedBy: user.ID}
		require.NoError(t, db.Create(&activeBucket).Error)

		deletedBucket := models.Bucket{Name: "deleted-bucket", CreatedBy: user.ID}
		require.NoError(t, db.Create(&deletedBucket).Error)

		activeBucketMembership := models.Membership{
			UserID:   user.ID,
			BucketID: activeBucket.ID,
			Group:    models.GroupOwner,
		}
		require.NoError(t, db.Create(&activeBucketMembership).Error)

		deletedBucketMembership := models.Membership{
			UserID:   user.ID,
			BucketID: deletedBucket.ID,
			Group:    models.GroupOwner,
		}
		require.NoError(t, db.Create(&deletedBucketMembership).Error)
		require.NoError(t, db.Delete(&deletedBucket).Error)

		for _, name := range []string{"first.txt", "second.txt"} {
			file := models.File{Name: name, Status: models.FileStatusUploaded, BucketID: activeBucket.ID, Size: 128}
			require.NoError(t, db.Create(&file).Error)
		}

		fileInDeletedBucket := models.File{
			Name:     "ghost.txt",
			Status:   models.FileStatusUploaded,
			BucketID: deletedBucket.ID,
			Size:     128,
		}
		require.NoError(t, db.Create(&fileInDeletedBucket).Error)

		deletedFile := models.File{
			Name:     "third.txt",
			Status:   models.FileStatusUploaded,
			BucketID: activeBucket.ID,
			Size:     128,
		}
		require.NoError(t, db.Create(&deletedFile).Error)
		require.NoError(t, db.Delete(&deletedFile).Error)

		response, err := service.GetUserStats(zap.NewNop(), models.UserClaims{}, []uuid.UUID{user.ID})
		require.NoError(t, err)
		require.Equal(t, models.UserStatsResponse{TotalFiles: 2, TotalBuckets: 1}, response)
	})

	t.Run("excludes files and buckets when membership is revoked", func(t *testing.T) {
		db := setupUserServiceTestDB(t)
		service := UserService{DB: db}

		user := models.User{
			Email:        "revoked-user@example.com",
			ProviderType: models.LocalProviderType,
			ProviderKey:  string(models.LocalProviderType),
			Role:         models.RoleUser,
		}
		require.NoError(t, db.Create(&user).Error)

		bucket := models.Bucket{Name: "shared-bucket", CreatedBy: user.ID}
		require.NoError(t, db.Create(&bucket).Error)

		membership := models.Membership{UserID: user.ID, BucketID: bucket.ID, Group: models.GroupOwner}
		require.NoError(t, db.Create(&membership).Error)

		for _, name := range []string{"a.txt", "b.txt"} {
			file := models.File{Name: name, Status: models.FileStatusUploaded, BucketID: bucket.ID, Size: 128}
			require.NoError(t, db.Create(&file).Error)
		}

		require.NoError(t, db.Delete(&membership).Error)

		response, err := service.GetUserStats(zap.NewNop(), models.UserClaims{}, []uuid.UUID{user.ID})
		require.NoError(t, err)
		require.Equal(t, models.UserStatsResponse{TotalFiles: 0, TotalBuckets: 0}, response)
	})
}
