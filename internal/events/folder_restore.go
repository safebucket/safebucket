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
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const FolderRestoreName = "FolderRestore"
const FolderRestorePayloadName = "FolderRestorePayload"

type FolderRestorePayload struct {
	Type     string
	BucketId uuid.UUID
	FolderId uuid.UUID
	UserId   uuid.UUID
}

type FolderRestore struct {
	Publisher messaging.IPublisher
	Payload   FolderRestorePayload
}

func NewFolderRestore(
	publisher messaging.IPublisher,
	bucketId uuid.UUID,
	folderId uuid.UUID,
	userId uuid.UUID,
) FolderRestore {
	return FolderRestore{
		Publisher: publisher,
		Payload: FolderRestorePayload{
			Type:     FolderRestoreName,
			BucketId: bucketId,
			FolderId: folderId,
			UserId:   userId,
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

func (e *FolderRestore) callback(params *EventParams) error {
	zap.L().Info("Starting folder restore",
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

	if folder.Status != models.FileStatusRestoring {
		zap.L().Warn("Folder not in restoring status, skipping",
			zap.String("current_status", string(folder.Status)))
		return nil
	}
	retentionPeriod := time.Duration(params.TrashRetentionDays) * 24 * time.Hour
	if folder.TrashedAt != nil && time.Since(*folder.TrashedAt) > retentionPeriod {
		zap.L().Error("Folder trash expired",
			zap.String("folder_id", folder.ID.String()),
			zap.Time("trashed_at", *folder.TrashedAt))

		params.DB.Model(&folder).Update("status", models.FileStatusTrashed)
		return errors.New("folder trash expired")
	}

	var existingFolder models.File
	conflictResult := params.DB.Where(
		"bucket_id = ? AND name = ? AND path = ? AND type = 'folder' AND (status IS NULL OR (status != ? AND status != ?))",
		e.Payload.BucketId, folder.Name, folder.Path, models.FileStatusTrashed, models.FileStatusRestoring,
	).First(&existingFolder)

	if conflictResult.RowsAffected > 0 {
		zap.L().Error("Folder name conflict detected",
			zap.String("folder_name", folder.Name),
			zap.String("path", folder.Path))

		params.DB.Model(&folder).Update("status", models.FileStatusTrashed)
		return errors.New("folder name conflict")
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":     models.FileStatusUploaded,
			"trashed_at": nil,
			"trashed_by": nil,
		}

		if err := tx.Model(&folder).Updates(updates).Error; err != nil {
			zap.L().Error("Failed to restore folder status", zap.Error(err))
			return err
		}

		objectPath := path.Join("buckets", e.Payload.BucketId.String(), folder.Path, folder.Name)
		if err := params.Storage.RemoveObjectTags(objectPath, []string{"Status", "TrashedAt"}); err != nil {
			zap.L().Error("Failed to remove trash tags from folder in storage - lifecycle policy may still target this folder",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("folder_id", e.Payload.FolderId.String()))
		}

		folderPath := path.Join(folder.Path, folder.Name)
		dbPath := fmt.Sprintf("%s%%", folderPath)

		var childFiles []models.File
		batchResult := tx.Where(
			"bucket_id = ? AND path LIKE ? AND status = ? AND trashed_at = ?",
			e.Payload.BucketId,
			dbPath,
			models.FileStatusTrashed,
			folder.TrashedAt,
		).Limit(c.BulkActionsLimit).Find(&childFiles)

		if batchResult.Error != nil {
			zap.L().Error("Failed to find child files for restore", zap.Error(batchResult.Error))
			return batchResult.Error
		}

		if len(childFiles) > 0 {
			zap.L().Info("Restoring folder contents batch",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var fileIds []uuid.UUID
			for _, child := range childFiles {
				fileIds = append(fileIds, child.ID)
			}

			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIds).
				Updates(updates).Error; err != nil {
				zap.L().Error("Failed to restore child files", zap.Error(err))
				return err
			}

			for _, child := range childFiles {
				childPath := path.Join("buckets", e.Payload.BucketId.String(), child.Path, child.Name)
				if err := params.Storage.RemoveObjectTags(childPath, []string{"Status", "TrashedAt"}); err != nil {
					zap.L().Error("Failed to remove trash tags from child object - lifecycle policy may still target this file",
						zap.Error(err),
						zap.String("path", childPath),
						zap.String("file_id", child.ID.String()))
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	var remainingCount int64
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND path LIKE ? AND status = ? AND trashed_at = ?",
		e.Payload.BucketId,
		fmt.Sprintf("%s%%", path.Join(folder.Path, folder.Name)),
		models.FileStatusTrashed,
		folder.TrashedAt,
	).Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More files to restore, requeuing event",
			zap.Int64("remaining", remainingCount))
		return errors.New("remaining files to restore")
	}

	action := models.Activity{
		Message: activity.FolderRestored,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   e.Payload.BucketId.String(),
			"file_id":     e.Payload.FolderId.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
			"user_id":     e.Payload.UserId.String(),
		}),
	}

	if err := params.ActivityLogger.Send(action); err != nil {
		zap.L().Error("Failed to log restore activity", zap.Error(err))
	}

	zap.L().Info("Folder restore complete",
		zap.String("bucket_id", e.Payload.BucketId.String()),
		zap.String("folder_id", e.Payload.FolderId.String()),
	)

	return nil
}
