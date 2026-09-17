package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

func TestGetUserList(t *testing.T) {
	t.Run("reports verified MFA without changing existing user fields", func(t *testing.T) {
		db := setupUserServiceTestDB(t)
		service := UserService{DB: db}
		users := []models.User{
			{
				FirstName: "Verified", LastName: "Admin", Email: "verified@example.com",
				ProviderType: models.LocalProviderType, ProviderKey: "password", Role: models.RoleAdmin,
				HashedPassword: "password-hash",
			},
			{
				FirstName: "Pending", LastName: "User", Email: "pending@example.com",
				ProviderType: models.OIDCProviderType, ProviderKey: "company-sso", Role: models.RoleUser,
			},
			{
				Email:        "no-devices@example.com",
				ProviderType: models.LDAPProviderType, ProviderKey: "directory", Role: models.RoleGuest,
			},
			{
				Email:        "deleted@example.com",
				ProviderType: models.LocalProviderType, ProviderKey: "password", Role: models.RoleUser,
			},
		}
		for i := range users {
			require.NoError(t, db.Create(&users[i]).Error)
		}
		for i, verifiedStates := range [][]bool{{true, true, false}, {false}, {}, {true}} {
			for j, verified := range verifiedStates {
				device := models.MFADevice{
					UserID: users[i].ID, Name: fmt.Sprintf("device-%d", j),
					Type: models.MFADeviceTypeTOTP, EncryptedSecret: "encrypted-secret", IsVerified: verified,
				}
				require.NoError(t, db.Create(&device).Error)
			}
		}
		require.NoError(t, db.Delete(&users[3]).Error)

		queries := 0
		require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:count_queries", func(_ *gorm.DB) {
			queries++
		}))

		result := service.GetUserList(zap.NewNop(), models.UserClaims{}, nil)
		require.Len(t, result, 3)
		require.Equal(t, 2, queries)

		byID := make(map[uuid.UUID]models.UserListItem, len(result))
		for _, item := range result {
			byID[item.ID] = item
		}
		require.NotContains(t, byID, users[3].ID)
		for i, user := range users[:3] {
			item, ok := byID[user.ID]
			require.True(t, ok)
			payload, err := json.Marshal(item)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(payload, &fields))
			require.Contains(t, fields, "mfa_enabled")
			require.JSONEq(t, strconv.FormatBool(i == 0), string(fields["mfa_enabled"]))
			delete(fields, "mfa_enabled")

			actual, err := json.Marshal(fields)
			require.NoError(t, err)
			expected, err := json.Marshal(user)
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(actual))
		}
	})

	t.Run("returns an empty array when there are no users", func(t *testing.T) {
		service := UserService{DB: setupUserServiceTestDB(t)}
		result := service.GetUserList(zap.NewNop(), models.UserClaims{}, nil)
		payload, err := json.Marshal(result)
		require.NoError(t, err)
		require.JSONEq(t, "[]", string(payload))
	})

	t.Run("logs MFA lookup failures and keeps the user list available", func(t *testing.T) {
		db := setupUserServiceTestDB(t)
		service := UserService{DB: db}
		user := models.User{
			Email:        "lookup-failure@example.com",
			ProviderType: models.LocalProviderType, ProviderKey: "password", Role: models.RoleUser,
		}
		require.NoError(t, db.Create(&user).Error)
		require.NoError(t, db.Migrator().DropTable(&models.MFADevice{}))
		core, logs := observer.New(zap.ErrorLevel)

		result := service.GetUserList(zap.New(core), models.UserClaims{}, nil)
		require.Len(t, result, 1)
		require.Equal(t, user.ID, result[0].ID)
		require.False(t, result[0].MFAEnabled)
		require.Equal(t, 1, logs.Len())
		require.Contains(t, logs.All()[0].ContextMap(), "error")
	})
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
