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

//nolint:gocognit,funlen // Complex event handler logic with multiple validation steps
func (e *FolderRestore) callback(params *EventParams) error {
	zap.L().Info("Starting folder restore",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	var childFiles []models.File

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		// Fetch folder inside transaction (use Unscoped to query soft-deleted folders)
		var folder models.Folder
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?",
			e.Payload.FolderID, e.Payload.BucketID).First(&folder)

		if result.Error != nil {
			zap.L().Error("Folder not found", zap.Error(result.Error))
			return result.Error
		}

		if folder.Status != models.FileStatusRestoring {
			zap.L().Warn("Folder not in restoring status, skipping",
				zap.String("current_status", string(folder.Status)))
			return nil
		}

		// Check for naming conflicts
		var existingFolder models.Folder
		query := tx.Where(
			"bucket_id = ? AND name = ? AND (status IS NULL OR (status != ? AND status != ?)) AND id != ?",
			e.Payload.BucketID,
			folder.Name,
			models.FileStatusTrashed,
			models.FileStatusRestoring,
			folder.ID,
		)
		if folder.FolderID != nil {
			query = query.Where("folder_id = ?", folder.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		conflictResult := query.Find(&existingFolder)

		if conflictResult.RowsAffected > 0 {
			zap.L().Error("Folder name conflict detected",
				zap.String("folder_name", folder.Name))

			if err := tx.Model(&folder).Update("status", models.FileStatusTrashed).Error; err != nil {
				zap.L().Error("Failed to revert folder status to trashed", zap.Error(err))
			}
			return errors.New("folder name conflict")
		}

		// Restore child folders (use Unscoped to query soft-deleted folders)
		var childFolders []models.Folder
		if err := tx.Unscoped().Where(
			"bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at IS NOT NULL",
			e.Payload.BucketID,
			e.Payload.FolderID,
			models.FileStatusTrashed,
		).Limit(c.BulkActionsLimit).Find(&childFolders).Error; err != nil {
			zap.L().Error("Failed to find child folders for restore", zap.Error(err))
			return err
		}

		if len(childFolders) > 0 {
			zap.L().Info("Restoring child folders",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFolders)))

			var folderIDs []uuid.UUID
			for _, child := range childFolders {
				folderIDs = append(folderIDs, child.ID)
			}

			// Clear soft delete and restore status
			if err := tx.Unscoped().Model(&models.Folder{}).
				Where("id IN ?", folderIDs).
				Updates(map[string]interface{}{
					"status":     nil,
					"deleted_at": nil,
					"trashed_by": nil,
				}).Error; err != nil {
				zap.L().Error("Failed to restore child folders", zap.Error(err))
				return err
			}
		}

		// Restore child files (use Unscoped to query soft-deleted files)
		if err := tx.Unscoped().Where(
			"bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at IS NOT NULL",
			e.Payload.BucketID,
			e.Payload.FolderID,
			models.FileStatusTrashed,
		).Limit(c.BulkActionsLimit).Find(&childFiles).Error; err != nil {
			zap.L().Error("Failed to find child files for restore", zap.Error(err))
			return err
		}

		if len(childFiles) > 0 {
			zap.L().Info("Restoring child files",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var fileIDs []uuid.UUID
			for _, child := range childFiles {
				fileIDs = append(fileIDs, child.ID)
			}

			// Update file status and clear soft delete FIRST, before unmarking from storage
			// This prevents race condition with trash_expiration handler
			if err := tx.Unscoped().Model(&models.File{}).
				Where("id IN ?", fileIDs).
				Updates(map[string]interface{}{
					"status":     models.FileStatusUploaded,
					"deleted_at": nil,
					"trashed_by": nil,
				}).Error; err != nil {
				zap.L().Error("Failed to restore child files", zap.Error(err))
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Unmark files from storage AFTER transaction commits
	// This ensures the file status is already updated when trash_expiration sees the marker deletion event
	if len(childFiles) > 0 {
		for _, child := range childFiles {
			filePath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
			if unmarkErr := params.Storage.UnmarkAsTrashed(filePath, child); unmarkErr != nil {
				zap.L().Warn("Failed to unmark file as trashed",
					zap.Error(unmarkErr),
					zap.String("file_id", child.ID.String()))
			}
		}
	}

	// Check if there are remaining items to restore (use Unscoped to query soft-deleted items)
	var remainingFolders int64
	params.DB.Unscoped().Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at IS NOT NULL",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusTrashed,
	).Count(&remainingFolders)

	var remainingFiles int64
	params.DB.Unscoped().Model(&models.File{}).Where(
		"bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at IS NOT NULL",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusTrashed,
	).Count(&remainingFiles)

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to restore, requeuing event",
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to restore")
	}

	// All children restored - now restore the folder itself
	err = params.DB.Transaction(func(tx *gorm.DB) error {
		var folder models.Folder
		// Use Unscoped to query soft-deleted folder
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?",
			e.Payload.FolderID, e.Payload.BucketID).First(&folder)

		if result.Error != nil {
			zap.L().Error("Folder not found for final restore", zap.Error(result.Error))
			return result.Error
		}

		if folder.Status != models.FileStatusRestoring {
			zap.L().Warn("Folder not in restoring status during final restore, skipping",
				zap.String("current_status", string(folder.Status)))
			// Already restored by another process, continue to log activity
			return nil
		}

		// Clear soft delete and restore status
		if updateErr := tx.Unscoped().Model(&folder).Updates(map[string]interface{}{
			"status":     nil,
			"deleted_at": nil,
			"trashed_by": nil,
		}).Error; updateErr != nil {
			zap.L().Error("Failed to restore folder status", zap.Error(updateErr))
			return updateErr
		}

		// Unmark folder from storage
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
		if unmarkErr := params.Storage.UnmarkAsTrashed(objectPath, folder); unmarkErr != nil {
			zap.L().Warn("Failed to unmark folder as trashed",
				zap.Error(unmarkErr),
				zap.String("path", objectPath),
				zap.String("folder_id", e.Payload.FolderID.String()))
			// Continue - folders exist only in DB
		}

		return nil
	})
	if err != nil {
		return err
	}

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

	if err = params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log restore activity", zap.Error(err))
	}

	zap.L().Info("Folder restore complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	return nil
}
