package services

import (
	"encoding/json"
	"testing"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newBucketServiceTestFixture(t *testing.T) (BucketService, models.User) {
	t.Helper()

	db := setupUserServiceTestDB(t)
	user := models.User{
		Email:        "bucket-service@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)

	service := BucketService{DB: db, ActivityLogger: &MockActivityLogger{}}
	return service, user
}

func createTestBucket(
	t *testing.T,
	service BucketService,
	user models.User,
	body models.BucketCreateUpdateBody,
) models.Bucket {
	t.Helper()

	bucket, err := service.CreateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{},
		body,
	)
	require.NoError(t, err)

	return bucket
}

func clearOutbox(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Where("1 = 1").Delete(&models.Outbox{}).Error)
}

func loadQueuedActivities(t *testing.T, db *gorm.DB) []models.Activity {
	t.Helper()

	var rows []models.Outbox
	require.NoError(t, db.Find(&rows).Error)

	activities := make([]models.Activity, 0, len(rows))
	for _, row := range rows {
		require.Equal(t, models.OutboxTypeActivity, row.Type)

		var entry models.Activity
		require.NoError(t, json.Unmarshal([]byte(row.Payload), &entry))
		activities = append(activities, entry)
	}

	return activities
}

func TestCreateBucketWithDescription(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "Team documentation"

	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})
	assert.Equal(t, "Team documentation", bucket.Description)

	var persisted models.Bucket
	require.NoError(t, service.DB.First(&persisted, "id = ?", bucket.ID).Error)
	assert.Equal(t, "Team documentation", persisted.Description)
}

func TestCreateBucketWithoutDescriptionDefaultsToEmpty(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)

	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{Name: "docs"})
	assert.Empty(t, bucket.Description)

	var persisted models.Bucket
	require.NoError(t, service.DB.First(&persisted, "id = ?", bucket.ID).Error)
	assert.Empty(t, persisted.Description)
}

func TestUpdateBucketUpdatesNameAndDescription(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{Name: "docs"})
	clearOutbox(t, service.DB)

	description := "Updated description"
	err := service.UpdateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID},
		models.BucketCreateUpdateBody{Name: "docs-renamed", Description: &description},
	)
	require.NoError(t, err)

	var persisted models.Bucket
	require.NoError(t, service.DB.First(&persisted, "id = ?", bucket.ID).Error)
	assert.Equal(t, "docs-renamed", persisted.Name)
	assert.Equal(t, "Updated description", persisted.Description)

	queued := loadQueuedActivities(t, service.DB)
	require.Len(t, queued, 1)
	assert.Equal(t, activity.BucketUpdated, queued[0].Message)
}

func TestUpdateBucketKeepsDescriptionWhenOmitted(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "Keep me"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})
	clearOutbox(t, service.DB)

	err := service.UpdateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID},
		models.BucketCreateUpdateBody{Name: "docs-renamed"},
	)
	require.NoError(t, err)

	var persisted models.Bucket
	require.NoError(t, service.DB.First(&persisted, "id = ?", bucket.ID).Error)
	assert.Equal(t, "docs-renamed", persisted.Name)
	assert.Equal(t, "Keep me", persisted.Description)

	queued := loadQueuedActivities(t, service.DB)
	require.Len(t, queued, 1)
	assert.Equal(t, activity.BucketUpdated, queued[0].Message)
}

func TestUpdateBucketClearsDescription(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "To be removed"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})
	clearOutbox(t, service.DB)

	empty := ""
	err := service.UpdateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID},
		models.BucketCreateUpdateBody{Name: "docs", Description: &empty},
	)
	require.NoError(t, err)

	var persisted models.Bucket
	require.NoError(t, service.DB.First(&persisted, "id = ?", bucket.ID).Error)
	assert.Empty(t, persisted.Description)

	queued := loadQueuedActivities(t, service.DB)
	require.Len(t, queued, 1)
	assert.Equal(t, activity.BucketUpdated, queued[0].Message)
}

func TestUpdateBucketDescriptionOnlyLogsBucketUpdatedActivity(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "Old description"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})
	clearOutbox(t, service.DB)

	updated := "New description"
	err := service.UpdateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID},
		models.BucketCreateUpdateBody{Name: "docs", Description: &updated},
	)
	require.NoError(t, err)

	queued := loadQueuedActivities(t, service.DB)
	require.Len(t, queued, 1)
	assert.Equal(t, activity.BucketUpdated, queued[0].Message)
	assert.Equal(t, bucket.ID.String(), queued[0].Filter.Fields.BucketID)

	object, ok := queued[0].Object.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, bucket.ID.String(), object["id"])
	assert.Equal(t, "docs", object["name"])
}
