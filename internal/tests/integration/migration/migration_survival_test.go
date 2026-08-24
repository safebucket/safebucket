//go:build integration

package migration_test

import (
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrationsPreserveData(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			dialect := bootstrap.LoadScenario(t, scenario).Database.Type

			versions, err := bootstrap.MigrationVersions(dialect)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(versions), 2, "need at least two migrations to test a forward migration")
			prev := versions[len(versions)-2]
			last := versions[len(versions)-1]

			db := bootstrap.NewDBProvider(t, dialect).Connect(t)
			sqlDB, err := db.DB()
			require.NoError(t, err)

			require.NoError(t, bootstrap.MigrateUpTo(sqlDB, dialect, prev), "migrate to penultimate version %d", prev)
			database.RegisterCallbacks(db)

			seedDB(t, db)

			require.NoError(t, bootstrap.MigrateUpTo(sqlDB, dialect, last), "migrate to latest version %d", last)

			for label, model := range map[string]any{
				"users":       &models.User{},
				"buckets":     &models.Bucket{},
				"memberships": &models.Membership{},
				"folders":     &models.Folder{},
				"files":       &models.File{},
				// One version row per file is the 00012 backfill; this is its only assertion
				// against a real Postgres.
				"file_versions": &models.FileVersion{},
				"invites":       &models.Invite{},
				"challenges":    &models.Challenge{},
				"mfa_devices":   &models.MFADevice{},
				"shares":        &models.Share{},
				"share_files":   &models.ShareFile{},
			} {
				var count int64
				require.NoError(t, db.Model(model).Count(&count).Error, "count %s", label)
				assert.Equal(t, int64(1), count, "%s row should survive migration to version %d", label, last)
			}
		})
	}
}

func seedDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	user := models.User{
		Email:        "user@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleAdmin,
	}
	require.NoError(t, db.Create(&user).Error)

	bucket := models.Bucket{Name: "bucket", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)

	require.NoError(t, db.Create(&models.Membership{
		UserID:   user.ID,
		BucketID: bucket.ID,
		Group:    models.GroupOwner,
	}).Error)

	folder := models.Folder{Name: "folder", Status: models.FolderStatusCreated, BucketID: bucket.ID}
	require.NoError(t, db.Create(&folder).Error)

	// Seeded column by column, not from models.File: the row is written against the penultimate
	// schema, while the model already carries the columns the migration under test adds, which a
	// struct create would reference in its insert and in GORM's RETURNING clause.
	fileID := uuid.New()
	require.NoError(t, db.Table("files").Create(map[string]any{
		"id":         fileID,
		"name":       "file.txt",
		"status":     models.FileStatusUploaded,
		"bucket_id":  bucket.ID,
		"folder_id":  folder.ID,
		"size":       10,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}).Error)

	invite := models.Invite{
		Email:     "invitee@example.com",
		Group:     models.GroupViewer,
		BucketID:  bucket.ID,
		CreatedBy: user.ID,
	}
	require.NoError(t, db.Create(&invite).Error)

	require.NoError(t, db.Create(&models.Challenge{
		Type:         models.ChallengeTypeInvite,
		HashedSecret: "secret",
		InviteID:     &invite.ID,
	}).Error)

	require.NoError(t, db.Create(&models.MFADevice{
		UserID:          user.ID,
		Name:            "device",
		Type:            models.MFADeviceTypeTOTP,
		EncryptedSecret: "encrypted",
	}).Error)

	share := models.Share{
		Name:      "share",
		BucketID:  bucket.ID,
		Type:      models.ShareTypeBucket,
		CreatedBy: user.ID,
	}
	require.NoError(t, db.Create(&share).Error)

	require.NoError(t, db.Create(&models.ShareFile{ShareID: share.ID, FileID: fileID}).Error)
}
