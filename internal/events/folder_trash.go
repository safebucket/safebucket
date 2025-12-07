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
	FolderTrashName        = "FolderTrash"
	FolderTrashPayloadName = "FolderTrashPayload"
)

type FolderTrashPayload struct {
	Type     string
	BucketID uuid.UUID
	FolderID uuid.UUID
	UserID   uuid.UUID
}

type FolderTrash struct {
	Publisher messaging.IPublisher
	Payload   FolderTrashPayload
}

func NewFolderTrash(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
) FolderTrash {
	return FolderTrash{
		Publisher: publisher,
		Payload: FolderTrashPayload{
			Type:     FolderTrashName,
			BucketID: bucketID,
			FolderID: folderID,
			UserID:   userID,
		},
	}
}

func (e *FolderTrash) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling folder trash event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger folder trash event", zap.Error(err))
	}
}

//nolint:gocognit // Complex event handler logic with multiple validation steps
func (e *FolderTrash) callback(params *EventParams) error {
	zap.L().Info("Starting folder trash",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	// Collect child folder IDs to trigger events after transaction
	var childFolderIDs []uuid.UUID
	var folderName string

	var folder models.Folder
	err := params.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?",
			e.Payload.FolderID, e.Payload.BucketID).First(&folder)

		if result.Error != nil {
			zap.L().Error("Folder not found", zap.Error(result.Error))
			return result.Error
		}

		folderName = folder.Name

		if !folder.DeletedAt.Valid {
			zap.L().Error("Folder not soft-deleted, cannot process children")
			return errors.New("folder not trashed")
		}

		var childFolders []models.Folder
		if err := tx.Where(
			"bucket_id = ? AND folder_id = ?",
			e.Payload.BucketID,
			e.Payload.FolderID,
		).Limit(c.BulkActionsLimit).Find(&childFolders).Error; err != nil {
			zap.L().Error("Failed to find child folders", zap.Error(err))
			return err
		}

		// Trash child folders (soft-delete them so their events can process)
		if len(childFolders) > 0 {
			zap.L().Info("Trashing child folders",
				zap.String("parent_folder", folder.Name),
				zap.String("parent_id", e.Payload.FolderID.String()),
				zap.Int("count", len(childFolders)))

			var folderIDs []uuid.UUID
			for _, child := range childFolders {
				zap.L().Debug("Child folder details",
					zap.String("child_id", child.ID.String()),
					zap.String("child_name", child.Name),
					zap.String("status", string(child.Status)))
				folderIDs = append(folderIDs, child.ID)
				childFolderIDs = append(childFolderIDs, child.ID)

				folderPath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
				if err := params.Storage.MarkAsTrashed(folderPath, child); err != nil {
					zap.L().Warn("Failed to mark child folder as trashed in storage",
						zap.Error(err),
						zap.String("folder_id", child.ID.String()))
				}
			}

			folderUpdates := map[string]interface{}{
				"status":     models.FileStatusDeleted,
				"deleted_by": e.Payload.UserID,
			}
			if err := tx.Model(&models.Folder{}).
				Where("id IN ?", folderIDs).
				Updates(folderUpdates).Error; err != nil {
				zap.L().Error("Failed to update child folders for trashing", zap.Error(err))
				return err
			}

			if err := tx.Where("id IN ?", folderIDs).Delete(&models.Folder{}).Error; err != nil {
				zap.L().Error("Failed to soft delete child folders", zap.Error(err))
				return err
			}
		} else {
			zap.L().Info("No child folders found",
				zap.String("parent_folder", folder.Name),
				zap.String("parent_id", e.Payload.FolderID.String()))
		}

		var childFiles []models.File
		if err := tx.Where(
			"bucket_id = ? AND folder_id = ?",
			e.Payload.BucketID,
			e.Payload.FolderID,
		).Limit(c.BulkActionsLimit).Find(&childFiles).Error; err != nil {
			zap.L().Error("Failed to find child files", zap.Error(err))
			return err
		}

		if len(childFiles) > 0 {
			zap.L().Info("Trashing child files",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var fileIDs []uuid.UUID
			for _, child := range childFiles {
				fileIDs = append(fileIDs, child.ID)

				filePath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
				if err := params.Storage.MarkAsTrashed(filePath, child); err != nil {
					zap.L().Warn("Failed to mark file as trashed in storage",
						zap.Error(err),
						zap.String("file_id", child.ID.String()))
				}
			}

			updates := map[string]interface{}{
				"status":     models.FileStatusDeleted,
				"deleted_by": e.Payload.UserID,
			}
			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIDs).
				Updates(updates).Error; err != nil {
				zap.L().Error("Failed to update child files for trashing", zap.Error(err))
				return err
			}

			if err := tx.Where("id IN ?", fileIDs).Delete(&models.File{}).Error; err != nil {
				zap.L().Error("Failed to soft delete child files", zap.Error(err))
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Trigger trash events for child folders after transaction commits
	if len(childFolderIDs) > 0 {
		zap.L().Info("Triggering trash events for child folders (after transaction commit)",
			zap.String("folder", folderName),
			zap.String("folder_id", e.Payload.FolderID.String()),
			zap.Int("child_count", len(childFolderIDs)))

		for _, childID := range childFolderIDs {
			zap.L().Info("Triggering child folder trash event",
				zap.String("parent_id", e.Payload.FolderID.String()),
				zap.String("child_id", childID.String()))

			childTrashEvent := NewFolderTrash(
				params.Publisher,
				e.Payload.BucketID,
				childID,
				e.Payload.UserID,
			)
			childTrashEvent.Trigger()
		}
	} else {
		zap.L().Info("No child folders to trigger events for",
			zap.String("folder", folderName),
			zap.String("folder_id", e.Payload.FolderID.String()))
	}

	// Check if there are remaining items to trash (items not yet soft-deleted)
	var remainingFolders int64
	params.DB.Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ?",
		e.Payload.BucketID,
		e.Payload.FolderID,
	).Count(&remainingFolders)

	var remainingFiles int64
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND folder_id = ?",
		e.Payload.BucketID,
		e.Payload.FolderID,
	).Count(&remainingFiles)

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to trash, requeuing event",
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to trash")
	}

	action := models.Activity{
		Message: activity.FolderTrashed,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionErase.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"folder_id":   e.Payload.FolderID.String(),
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     e.Payload.UserID.String(),
		}),
	}

	if err = params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log trash activity", zap.Error(err))
	}

	zap.L().Info("Folder trash complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	return nil
}
