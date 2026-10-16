package services

import (
	"testing"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

	mock := service.ActivityLogger.(*MockActivityLogger)
	assert.Equal(t, activity.BucketUpdated, mock.Sent[len(mock.Sent)-1].Message)
}

func TestUpdateBucketKeepsDescriptionWhenOmitted(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "Keep me"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})

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

	mock := service.ActivityLogger.(*MockActivityLogger)
	assert.Equal(t, activity.BucketUpdated, mock.Sent[len(mock.Sent)-1].Message)
}

func TestUpdateBucketClearsDescription(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "To be removed"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})

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

	mock := service.ActivityLogger.(*MockActivityLogger)
	assert.Equal(t, activity.BucketDescriptionUpdated, mock.Sent[len(mock.Sent)-1].Message)
}

func TestUpdateBucketDescriptionOnlyLogsDescriptionUpdatedActivity(t *testing.T) {
	service, user := newBucketServiceTestFixture(t)
	description := "Old description"
	bucket := createTestBucket(t, service, user, models.BucketCreateUpdateBody{
		Name:        "docs",
		Description: &description,
	})

	mock := service.ActivityLogger.(*MockActivityLogger)
	mock.Sent = nil

	updated := "New description"
	err := service.UpdateBucket(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID},
		models.BucketCreateUpdateBody{Name: "docs", Description: &updated},
	)
	require.NoError(t, err)

	require.Len(t, mock.Sent, 1)
	assert.Equal(t, activity.BucketDescriptionUpdated, mock.Sent[0].Message)
	assert.Equal(t, bucket.ID.String(), mock.Sent[0].Filter.Fields.BucketID)

	object, ok := mock.Sent[0].Object.(models.BucketActivity)
	require.True(t, ok)
	assert.Equal(t, bucket.ID, object.ID)
	assert.Equal(t, "docs", object.Name)
}
