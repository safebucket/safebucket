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
	FolderPurgeName        = "FolderPurge"
	FolderPurgePayloadName = "FolderPurgePayload"
)

type FolderPurgePayload struct {
	Type     string
	BucketID uuid.UUID
	FolderID uuid.UUID
	UserID   uuid.UUID
}

type FolderPurge struct {
	Publisher messaging.IPublisher
	Payload   FolderPurgePayload
}

func NewFolderPurge(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
) FolderPurge {
	return FolderPurge{
		Publisher: publisher,
		Payload: FolderPurgePayload{
			Type:     FolderPurgeName,
			BucketID: bucketID,
			FolderID: folderID,
			UserID:   userID,
		},
	}
}

func (e *FolderPurge) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling folder purge event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger folder purge event", zap.Error(err))
	}
}

func (e *FolderPurge) callback(params *EventParams) error {
	zap.L().Info("Starting folder purge (permanent deletion)",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	var folder models.Folder
	result := params.DB.Unscoped().Where("id = ? AND bucket_id = ?",
		e.Payload.FolderID, e.Payload.BucketID).First(&folder)

	if result.Error != nil {
		zap.L().Error("Folder not found", zap.Error(result.Error))
		return result.Error
	}

	if !folder.DeletedAt.Valid {
		zap.L().Warn("Folder not soft-deleted, cannot purge")
		return errors.New("folder not in trash")
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		var childFolders []models.Folder
		if err := tx.Unscoped().Where(
			"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL",
			e.Payload.BucketID,
			e.Payload.FolderID,
		).Limit(c.BulkActionsLimit).Find(&childFolders).Error; err != nil {
			zap.L().Error("Failed to find child folders for purging", zap.Error(err))
			return err
		}

		if len(childFolders) > 0 {
			zap.L().Info("Purging child folders",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFolders)))

			var folderIDs []uuid.UUID
			for _, child := range childFolders {
				folderIDs = append(folderIDs, child.ID)
			}

			if err := tx.Unscoped().Where("id IN ?", folderIDs).Delete(&models.Folder{}).Error; err != nil {
				zap.L().Error("Failed to hard delete child folders", zap.Error(err))
				return err
			}
		}

		var childFiles []models.File
		if err := tx.Unscoped().Where(
			"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL",
			e.Payload.BucketID,
			e.Payload.FolderID,
		).Limit(c.BulkActionsLimit).Find(&childFiles).Error; err != nil {
			zap.L().Error("Failed to find child files for purging", zap.Error(err))
			return err
		}

		if len(childFiles) > 0 {
			zap.L().Info("Purging child files",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var storagePaths []string
			var fileIDs []uuid.UUID
			for _, child := range childFiles {
				fileIDs = append(fileIDs, child.ID)

				childPath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
				storagePaths = append(storagePaths, childPath)
			}

			if len(storagePaths) > 0 {
				if err := params.Storage.RemoveObjects(storagePaths); err != nil {
					zap.L().Warn("Failed to delete some files from storage", zap.Error(err))
				}
			}

			if err := tx.Unscoped().Where("id IN ?", fileIDs).Delete(&models.File{}).Error; err != nil {
				zap.L().Error("Failed to hard delete child files", zap.Error(err))
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	var remainingFolders int64
	params.DB.Unscoped().Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL",
		e.Payload.BucketID,
		e.Payload.FolderID,
	).Count(&remainingFolders)

	var remainingFiles int64
	params.DB.Unscoped().Model(&models.File{}).Where(
		"bucket_id = ? AND folder_id = ? AND deleted_at IS NOT NULL",
		e.Payload.BucketID,
		e.Payload.FolderID,
	).Count(&remainingFiles)

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to purge, requeuing event",
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to purge")
	}

	objectPath := path.Join("buckets", e.Payload.BucketID.String(), e.Payload.FolderID.String())
	folderModel := models.Folder{ID: e.Payload.FolderID, BucketID: e.Payload.BucketID}
	if err = params.Storage.UnmarkAsTrashed(objectPath, folderModel); err != nil {
		zap.L().Warn("Failed to delete folder marker from storage",
			zap.Error(err),
			zap.String("folder_id", e.Payload.FolderID.String()))
	}

	if err = params.DB.Unscoped().Delete(&folder).Error; err != nil {
		zap.L().Error("Failed to hard delete folder from database", zap.Error(err))
		return err
	}

	action := models.Activity{
		Message: activity.FolderDeleted,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDelete.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"folder_id":   e.Payload.FolderID.String(),
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     e.Payload.UserID.String(),
		}),
	}

	if err = params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log purge activity", zap.Error(err))
	}

	zap.L().Info("Folder purge complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	return nil
}
