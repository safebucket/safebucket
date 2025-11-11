package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"time"

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
	Type      string
	BucketID  uuid.UUID
	FolderID  uuid.UUID
	UserID    uuid.UUID
	TrashedAt time.Time
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
	trashedAt time.Time,
) FolderTrash {
	return FolderTrash{
		Publisher: publisher,
		Payload: FolderTrashPayload{
			Type:      FolderTrashName,
			BucketID:  bucketID,
			FolderID:  folderID,
			UserID:    userID,
			TrashedAt: trashedAt,
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

func (e *FolderTrash) callback(params *EventParams) error {
	zap.L().Info("Starting folder trash",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)

	var folder models.File
	result := params.DB.Where("id = ? AND bucket_id = ? AND type = 'folder'",
		e.Payload.FolderID, e.Payload.BucketID).First(&folder)

	if result.Error != nil {
		zap.L().Error("Folder not found", zap.Error(result.Error))
		return result.Error
	}

	if folder.Status != models.FileStatusTrashed {
		zap.L().Warn("Folder not in trashed status, skipping",
			zap.String("current_status", string(folder.Status)))
		return nil
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.Path, folder.Name)
		if err := params.Storage.MarkFileAsTrashed(objectPath, models.TrashMetadata{
			OriginalPath: objectPath,
			TrashedAt:    e.Payload.TrashedAt,
			TrashedBy:    e.Payload.UserID,
			FileID:       e.Payload.FolderID,
			IsFolder:     true,
		}); err != nil {
			zap.L().
				Error("Failed to mark folder as trashed - rolling back transaction",
					zap.Error(err),
					zap.String("path", objectPath),
					zap.String("folder_id", e.Payload.FolderID.String()))
			return err
		}

		folderPath := path.Join(folder.Path, folder.Name)
		dbPath := fmt.Sprintf("%s/%%", folderPath)

		var childFiles []models.File
		batchResult := tx.Where(
			"bucket_id = ? AND path LIKE ? AND status != ?",
			e.Payload.BucketID,
			dbPath,
			models.FileStatusTrashed,
		).Limit(c.BulkActionsLimit).Find(&childFiles)

		if batchResult.Error != nil {
			zap.L().Error("Failed to find child files for trashing", zap.Error(batchResult.Error))
			return batchResult.Error
		}

		if len(childFiles) > 0 {
			zap.L().Info("Trashing folder contents batch",
				zap.String("folder", folder.Name),
				zap.Int("child_count", len(childFiles)))

			var fileIDs []uuid.UUID
			for _, child := range childFiles {
				fileIDs = append(fileIDs, child.ID)
			}

			updates := map[string]interface{}{
				"status":     models.FileStatusTrashed,
				"trashed_at": e.Payload.TrashedAt,
				"trashed_by": e.Payload.UserID,
			}

			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIDs).
				Updates(updates).Error; err != nil {
				zap.L().Error("Failed to trash child files", zap.Error(err))
				return err
			}

			// Note: We only mark the folder itself in storage, not individual child files.
			zap.L().Debug("Child files marked as trashed in database only",
				zap.Int("child_count", len(childFiles)))
		}

		return nil
	})
	if err != nil {
		return err
	}

	var remainingCount int64
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND path LIKE ? AND status != ?",
		e.Payload.BucketID,
		fmt.Sprintf("%s/%%", path.Join(folder.Path, folder.Name)),
		models.FileStatusTrashed,
	).Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More files to trash, requeuing event",
			zap.Int64("remaining", remainingCount))
		return errors.New("remaining files to trash")
	}

	action := models.Activity{
		Message: activity.FolderTrashed,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionErase.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"file_id":     e.Payload.FolderID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
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
