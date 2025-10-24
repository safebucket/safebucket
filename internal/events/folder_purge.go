package events

import (
	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/rbac"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const FolderPurgeName = "FolderPurge"
const FolderPurgePayloadName = "FolderPurgePayload"

type FolderPurgePayload struct {
	Type     string
	BucketId uuid.UUID
	FolderId uuid.UUID
	UserId   uuid.UUID
}

type FolderPurge struct {
	Publisher messaging.IPublisher
	Payload   FolderPurgePayload
}

func NewFolderPurge(
	publisher messaging.IPublisher,
	bucketId uuid.UUID,
	folderId uuid.UUID,
	userId uuid.UUID,
) FolderPurge {
	return FolderPurge{
		Publisher: publisher,
		Payload: FolderPurgePayload{
			Type:     FolderPurgeName,
			BucketId: bucketId,
			FolderId: folderId,
			UserId:   userId,
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
		zap.String("bucket_id", e.Payload.BucketId.String()),
		zap.String("folder_id", e.Payload.FolderId.String()),
	)

	var folder models.File
	result := params.DB.Where("id = ? AND bucket_id = ? AND type = 'folder'",
		e.Payload.FolderId, e.Payload.BucketId).First(&folder)

	if result.Error != nil {
		zap.L().Error("Folder not found", zap.Error(result.Error))
		return result.Error
	}

	if folder.Status != models.FileStatusTrashed {
		zap.L().Warn("Folder not in trashed status, cannot purge",
			zap.String("current_status", string(folder.Status)))
		return errors.New("folder not in trash")
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		folderPath := path.Join(folder.Path, folder.Name)
		dbPath := fmt.Sprintf("%s%%", folderPath)

		var childFiles []models.File
		batchResult := tx.Where(
			"bucket_id = ? AND path LIKE ?",
			e.Payload.BucketId,
			dbPath,
		).Limit(c.BulkActionsLimit).Find(&childFiles)

		if batchResult.Error != nil {
			zap.L().Error("Failed to find child files for purging", zap.Error(batchResult.Error))
			return batchResult.Error
		}

		if len(childFiles) > 0 {
			zap.L().Info("Purging folder contents batch",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var storagePaths []string
			for _, child := range childFiles {
				childPath := path.Join("buckets", e.Payload.BucketId.String(), child.Path, child.Name)
				storagePaths = append(storagePaths, childPath)
			}

			if len(storagePaths) > 0 {
				err := params.Storage.RemoveObjects(storagePaths)
				if err != nil {
					zap.L().Error("Failed to delete files from storage", zap.Error(err))
				}
			}

			var fileIds []uuid.UUID
			for _, child := range childFiles {
				fileIds = append(fileIds, child.ID)
			}

			if err := tx.Where("id IN ?", fileIds).Delete(&models.File{}).Error; err != nil {
				zap.L().Error("Failed to soft delete child files", zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	var remainingCount int64
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND path LIKE ?",
		e.Payload.BucketId,
		fmt.Sprintf("%s%%", path.Join(folder.Path, folder.Name)),
	).Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More files to purge, requeuing event",
			zap.Int64("remaining", remainingCount))
		return errors.New("remaining files to purge")
	}

	objectPath := path.Join("buckets", e.Payload.BucketId.String(), folder.Path, folder.Name)
	if err := params.Storage.RemoveObject(objectPath); err != nil {
		zap.L().Warn("Failed to delete folder from storage",
			zap.Error(err),
			zap.String("path", objectPath))
	}

	if err := params.DB.Delete(&folder).Error; err != nil {
		zap.L().Error("Failed to soft delete folder from database", zap.Error(err))
		return err
	}

	action := models.Activity{
		Message: activity.FolderPurged,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionPurge.String(),
			"bucket_id":   e.Payload.BucketId.String(),
			"file_id":     e.Payload.FolderId.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
			"user_id":     e.Payload.UserId.String(),
		}),
	}

	if err := params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log purge activity", zap.Error(err))
	}

	zap.L().Info("Folder purge complete",
		zap.String("bucket_id", e.Payload.BucketId.String()),
		zap.String("folder_id", e.Payload.FolderId.String()),
	)

	return nil
}
