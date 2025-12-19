package helpers

import (
	apierrors "api/internal/errors"
	"api/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RestoreParentFolders restores all trashed parent folders in the hierarchy.
// Returns the list of restored folders so their trash markers can be removed after commit.
func RestoreParentFolders(
	tx *gorm.DB,
	logger *zap.Logger,
	folderID *uuid.UUID,
	bucketID uuid.UUID,
) ([]models.Folder, error) {
	if folderID == nil {
		return nil, nil
	}

	var trashedFolderIDs []uuid.UUID
	currentFolderID := folderID

	for currentFolderID != nil {
		var folder models.Folder
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?", currentFolderID, bucketID).First(&folder)
		if result.Error != nil {
			break
		}

		if folder.DeletedAt.Valid {
			trashedFolderIDs = append(trashedFolderIDs, folder.ID)
		}

		currentFolderID = folder.FolderID
	}

	if len(trashedFolderIDs) == 0 {
		return nil, nil
	}

	var trashedFolders []models.Folder
	result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", trashedFolderIDs).
		Find(&trashedFolders)
	if result.Error != nil {
		return nil, result.Error
	}

	folderMap := make(map[uuid.UUID]models.Folder)
	for _, f := range trashedFolders {
		folderMap[f.ID] = f
	}

	var restoredFolders []models.Folder

	for i := len(trashedFolderIDs) - 1; i >= 0; i-- {
		folder, exists := folderMap[trashedFolderIDs[i]]
		if !exists {
			continue
		}

		if folder.Status == models.FileStatusRestoring || !folder.DeletedAt.Valid {
			continue
		}

		var existingFolder models.Folder
		query := tx.Where(
			"bucket_id = ? AND name = ? AND id != ?",
			folder.BucketID, folder.Name, folder.ID,
		)
		if folder.FolderID != nil {
			query = query.Where("folder_id = ?", folder.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		if query.Find(&existingFolder); existingFolder.ID != uuid.Nil {
			return nil, apierrors.NewAPIError(409, "PARENT_FOLDER_NAME_CONFLICT")
		}

		updates := map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
			"status":     nil,
		}
		if err := tx.Unscoped().Model(&folder).Updates(updates).Error; err != nil {
			logger.Error("Failed to restore parent folder",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
			return nil, err
		}

		restoredFolders = append(restoredFolders, folder)

		logger.Info("Restored parent folder",
			zap.String("folder_name", folder.Name),
			zap.String("folder_id", folder.ID.String()))
	}

	return restoredFolders, nil
}
