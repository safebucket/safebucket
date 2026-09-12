package services

import (
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildFolderPath(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	foldersByID := map[uuid.UUID]models.Folder{
		parentID: {ID: parentID, Name: "parent"},
		childID:  {ID: childID, Name: "child", FolderID: &parentID},
	}

	assert.Equal(t, "/", buildFolderPath(nil, foldersByID))
	assert.Equal(t, "/parent/child", buildFolderPath(&childID, foldersByID))
	assert.Equal(t, "/", buildFolderPath(new(uuid.UUID), foldersByID))
}

func TestPurgeFolderRejectsRestoringFolder(t *testing.T) {
	db := setupUserServiceTestDB(t)
	user := models.User{
		Email:        "folder-purge@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)

	bucket := models.Bucket{Name: "folder-purge", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)

	folder := models.Folder{
		Name:     "restoring",
		Status:   models.FolderStatusRestoring,
		BucketID: bucket.ID,
	}
	require.NoError(t, db.Create(&folder).Error)
	require.NoError(t, db.Delete(&folder).Error)

	service := BucketFolderService{DB: db}
	err := service.DeleteFolder(
		zap.NewNop(),
		models.UserClaims{UserID: user.ID},
		uuid.UUIDs{bucket.ID, folder.ID},
	)

	var apiErr *apierrors.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierrors.CodeFolderRestoreInProgress, apiErr.Code)

	var persisted models.Folder
	require.NoError(t, db.Unscoped().First(&persisted, "id = ?", folder.ID).Error)
	assert.True(t, persisted.DeletedAt.Valid)
	assert.Equal(t, models.FolderStatusRestoring, persisted.Status)
}
