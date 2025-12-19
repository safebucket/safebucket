package events

import (
	"encoding/json"
	"errors"
	"path"

	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/rbac"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FolderRestoreName        = "FolderRestore"
	FolderRestorePayloadName = "FolderRestorePayload"
)

type FolderRestorePayload struct {
	Type     string
	BucketID uuid.UUID
	FolderID uuid.UUID
	UserID   uuid.UUID
}

type FolderRestore struct {
	Publisher messaging.IPublisher
	Payload   FolderRestorePayload
}

func NewFolderRestore(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
) FolderRestore {
	return FolderRestore{
		Publisher: publisher,
		Payload: FolderRestorePayload{
			Type:     FolderRestoreName,
			BucketID: bucketID,
			FolderID: folderID,
			UserID:   userID,
		},
	}
}

func (e *FolderRestore) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling folder restore event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger folder restore event", zap.Error(err))
	}
}

// restoreState holds intermediate state during folder restore processing.
type restoreState struct {
	folderName     string
	childFolderIDs []uuid.UUID
	childFiles     []models.File
}

func (e *FolderRestore) callback(params *EventParams) error {
	zap.L().Info("Starting folder restore",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	state, err := e.processChildrenInTransaction(params)
	if err != nil {
		return err
	}

	e.triggerChildFolderRestoreEvents(params, state)
	e.unmarkChildFilesFromStorage(params, state.childFiles)

	if err = e.checkRemainingItemsToRestore(params); err != nil {
		return err
	}

	if err = e.checkChildFoldersStillRestoring(params); err != nil {
		return err
	}

	return e.finalizeFolderRestore(params)
}

func (e *FolderRestore) processChildrenInTransaction(params *EventParams) (*restoreState, error) {
	state := &restoreState{}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		folder, txErr := e.fetchAndValidateFolder(tx)
		if txErr != nil {
			return txErr
		}
		if folder == nil {
			return nil
		}
		state.folderName = folder.Name

		if txErr = e.checkFolderNameConflict(tx, folder); txErr != nil {
			return txErr
		}

		childFolderIDs, txErr := e.processChildFolders(tx, folder.Name)
		if txErr != nil {
			return txErr
		}
		state.childFolderIDs = childFolderIDs

		childFiles, txErr := e.processChildFiles(tx, folder.Name)
		if txErr != nil {
			return txErr
		}
		state.childFiles = childFiles

		return nil
	})

	return state, err
}

func (e *FolderRestore) fetchAndValidateFolder(tx *gorm.DB) (*models.Folder, error) {
	var folder models.Folder
	result := tx.Unscoped().Where("id = ? AND bucket_id = ?",
		e.Payload.FolderID, e.Payload.BucketID).First(&folder)

	if result.Error != nil {
		zap.L().Error("Folder not found", zap.Error(result.Error))
		return nil, result.Error
	}

	if folder.Status != models.FileStatusRestoring {
		zap.L().Warn("Folder not in restoring status, skipping",
			zap.String("current_status", string(folder.Status)))
		return nil, nil
	}

	return &folder, nil
}

func (e *FolderRestore) checkFolderNameConflict(tx *gorm.DB, folder *models.Folder) error {
	var existingFolder models.Folder
	query := tx.Where(
		"bucket_id = ? AND name = ? AND id != ?",
		e.Payload.BucketID,
		folder.Name,
		folder.ID,
	)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}

	if query.Find(&existingFolder); existingFolder.ID != uuid.Nil {
		zap.L().Error("Folder name conflict detected", zap.String("folder_name", folder.Name))
		if err := tx.Unscoped().Model(folder).Update("status", models.FileStatusDeleted).Error; err != nil {
			zap.L().Error("Failed to revert folder status", zap.Error(err))
		}
		return errors.New("folder name conflict")
	}

	return nil
}

func (e *FolderRestore) processChildFolders(tx *gorm.DB, parentName string) ([]uuid.UUID, error) {
	var childFolders []models.Folder
	if err := tx.Unscoped().Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status = ?)",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusDeleted,
	).Limit(c.BulkActionsLimit).Find(&childFolders).Error; err != nil {
		zap.L().Error("Failed to find child folders for restore", zap.Error(err))
		return nil, err
	}

	if len(childFolders) == 0 {
		zap.L().Info("No child folders found to restore",
			zap.String("parent_folder", parentName),
			zap.String("parent_id", e.Payload.FolderID.String()))
		return nil, nil
	}

	zap.L().Info("Setting child folders to restoring status",
		zap.String("parent_folder", parentName),
		zap.String("parent_id", e.Payload.FolderID.String()),
		zap.Int("count", len(childFolders)))

	var folderIDs []uuid.UUID
	for _, child := range childFolders {
		zap.L().Debug("Child folder details",
			zap.String("child_id", child.ID.String()),
			zap.String("child_name", child.Name),
			zap.String("status", string(child.Status)))
		folderIDs = append(folderIDs, child.ID)
	}

	if err := tx.Unscoped().Model(&models.Folder{}).
		Where("id IN ?", folderIDs).
		Update("status", models.FileStatusRestoring).Error; err != nil {
		zap.L().Error("Failed to set child folders to restoring status", zap.Error(err))
		return nil, err
	}

	return folderIDs, nil
}

func (e *FolderRestore) processChildFiles(tx *gorm.DB, parentName string) ([]models.File, error) {
	var childFiles []models.File
	if err := tx.Unscoped().Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status = ?)",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusDeleted,
	).Limit(c.BulkActionsLimit).Find(&childFiles).Error; err != nil {
		zap.L().Error("Failed to find child files for restore", zap.Error(err))
		return nil, err
	}

	if len(childFiles) == 0 {
		return nil, nil
	}

	zap.L().Info("Restoring child files",
		zap.String("folder", parentName),
		zap.Int("child_count", len(childFiles)))

	var fileIDs []uuid.UUID
	for _, child := range childFiles {
		fileIDs = append(fileIDs, child.ID)
	}

	if err := tx.Unscoped().Model(&models.File{}).
		Where("id IN ?", fileIDs).
		Update("status", models.FileStatusRestoring).Error; err != nil {
		zap.L().Error("Failed to set child files to restoring status", zap.Error(err))
		return nil, err
	}

	if err := tx.Unscoped().Model(&models.File{}).
		Where("id IN ?", fileIDs).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
			"status":     models.FileStatusUploaded,
		}).Error; err != nil {
		zap.L().Error("Failed to restore child files", zap.Error(err))
		return nil, err
	}

	return childFiles, nil
}

func (e *FolderRestore) triggerChildFolderRestoreEvents(params *EventParams, state *restoreState) {
	if len(state.childFolderIDs) == 0 {
		zap.L().Info("No child folders to trigger events for",
			zap.String("folder", state.folderName),
			zap.String("folder_id", e.Payload.FolderID.String()))
		return
	}

	zap.L().Info("Triggering restore events for child folders (after transaction commit)",
		zap.String("folder", state.folderName),
		zap.String("folder_id", e.Payload.FolderID.String()),
		zap.Int("child_count", len(state.childFolderIDs)))

	for _, childID := range state.childFolderIDs {
		zap.L().Info("Triggering child folder restore event",
			zap.String("parent_id", e.Payload.FolderID.String()),
			zap.String("child_id", childID.String()))

		childRestoreEvent := NewFolderRestore(
			params.Publisher,
			e.Payload.BucketID,
			childID,
			e.Payload.UserID,
		)
		childRestoreEvent.Trigger()
	}
}

func (e *FolderRestore) unmarkChildFilesFromStorage(params *EventParams, childFiles []models.File) {
	for _, child := range childFiles {
		filePath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
		if err := params.Storage.UnmarkAsTrashed(filePath, child); err != nil {
			zap.L().Warn("Failed to unmark file as trashed",
				zap.Error(err),
				zap.String("file_id", child.ID.String()))
		}
	}
}

func (e *FolderRestore) checkRemainingItemsToRestore(params *EventParams) error {
	var remainingFolders int64
	params.DB.Unscoped().Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status = ?)",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusDeleted,
	).Count(&remainingFolders)

	var remainingFiles int64
	params.DB.Unscoped().Model(&models.File{}).Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status = ?)",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusDeleted,
	).Count(&remainingFiles)

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to restore, requeuing event",
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to restore")
	}

	return nil
}

func (e *FolderRestore) checkChildFoldersStillRestoring(params *EventParams) error {
	var restoringFolders int64
	params.DB.Unscoped().Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ? AND status = ?",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusRestoring,
	).Count(&restoringFolders)

	if restoringFolders > 0 {
		zap.L().Info("Child folders still restoring, requeuing event",
			zap.Int64("restoring_folders", restoringFolders))
		return errors.New("child folders still restoring")
	}

	return nil
}

func (e *FolderRestore) finalizeFolderRestore(params *EventParams) error {
	var folder models.Folder

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?",
			e.Payload.FolderID, e.Payload.BucketID).First(&folder)

		if result.Error != nil {
			zap.L().Error("Folder not found for final restore", zap.Error(result.Error))
			return result.Error
		}

		if folder.Status != models.FileStatusRestoring {
			zap.L().Warn("Folder not in restoring status during final restore, skipping",
				zap.String("current_status", string(folder.Status)))
			return nil
		}

		if err := tx.Unscoped().Model(&folder).Updates(map[string]interface{}{
			"status":     nil,
			"deleted_at": nil,
			"deleted_by": nil,
		}).Error; err != nil {
			zap.L().Error("Failed to restore folder status", zap.Error(err))
			return err
		}

		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
		if err := params.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			zap.L().Warn("Failed to unmark folder as trashed",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("folder_id", e.Payload.FolderID.String()))
		}

		return nil
	})
	if err != nil {
		return err
	}

	e.logRestoreActivity(params, &folder)

	zap.L().Debug("Folder restore complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	return nil
}

func (e *FolderRestore) logRestoreActivity(params *EventParams, folder *models.Folder) {
	action := models.Activity{
		Message: activity.FolderRestored,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"folder_id":   e.Payload.FolderID.String(),
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     e.Payload.UserID.String(),
		}),
	}

	if err := params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log restore activity", zap.Error(err))
	}
}
