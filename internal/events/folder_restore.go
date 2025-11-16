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

func (e *FolderRestore) callback(params *EventParams) error {
	zap.L().Info("Starting folder restore",
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
		e.Payload.BucketID,
		folder.Name,
		folder.Path,
		models.FileStatusTrashed,
		models.FileStatusRestoring,
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

		folderPath := path.Join(folder.Path, folder.Name)
		dbPath := fmt.Sprintf("%s/%%", folderPath)

		var childFiles []models.File
		batchResult := tx.Where(
			"bucket_id = ? AND (path = ? OR path LIKE ?) AND status = ? AND trashed_at = ?",
			e.Payload.BucketID,
			folderPath,
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

			for _, child := range childFiles {
				var existingFile models.File
				childConflictResult := tx.Where(
					"bucket_id = ? AND name = ? AND path = ? AND status IS NOT NULL AND status != ? AND status != ?",
					e.Payload.BucketID,
					child.Name,
					child.Path,
					models.FileStatusTrashed,
					models.FileStatusRestoring,
				).First(&existingFile)

				if childConflictResult.RowsAffected > 0 {
					zap.L().Error("Child file name conflict detected, aborting restore",
						zap.String("file_name", child.Name),
						zap.String("path", child.Path),
						zap.String("conflicting_file_id", existingFile.ID.String()))

					tx.Model(&folder).Update("status", models.FileStatusTrashed)
					return errors.New("child file name conflict")
				}
			}

			var fileIDs []uuid.UUID
			for _, child := range childFiles {
				fileIDs = append(fileIDs, child.ID)
			}

			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIDs).
				Updates(updates).Error; err != nil {
				zap.L().Error("Failed to restore child files", zap.Error(err))
				return err
			}

			zap.L().Debug("Child files restored in database only",
				zap.Int("child_count", len(childFiles)))
		}

		return nil
	})
	if err != nil {
		return err
	}

	objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.Path, folder.Name)
	if unmarkErr := params.Storage.UnmarkFileAsTrashed(objectPath); unmarkErr != nil {
		zap.L().
			Error("Failed to unmark folder as trashed",
				zap.Error(unmarkErr),
				zap.String("path", objectPath),
				zap.String("folder_id", e.Payload.FolderID.String()))
		revertUpdates := map[string]interface{}{
			"status":     models.FileStatusTrashed,
			"trashed_at": folder.TrashedAt,
			"trashed_by": folder.TrashedBy,
		}
		if revertErr := params.DB.Model(&folder).Updates(revertUpdates).Error; revertErr != nil {
			zap.L().Error("Failed to revert folder status after storage error",
				zap.Error(revertErr),
				zap.String("folder_id", e.Payload.FolderID.String()))
		}
		return unmarkErr
	}

	var remainingCount int64
	folderPathForCount := path.Join(folder.Path, folder.Name)
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND (path = ? OR path LIKE ?) AND status = ? AND trashed_at = ?",
		e.Payload.BucketID,
		folderPathForCount,
		fmt.Sprintf("%s/%%", folderPathForCount),
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
			"bucket_id":   e.Payload.BucketID.String(),
			"file_id":     e.Payload.FolderID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
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
