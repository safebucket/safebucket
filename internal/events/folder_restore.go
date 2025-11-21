package events

import (
	"encoding/json"
	"errors"
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

	var folder models.Folder
	result := params.DB.Where("id = ? AND bucket_id = ?",
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

	// Check for naming conflicts
	var existingFolder models.Folder
	query := params.DB.Where(
		"bucket_id = ? AND name = ? AND (status IS NULL OR (status != ? AND status != ?))",
		e.Payload.BucketID,
		folder.Name,
		models.FileStatusTrashed,
		models.FileStatusRestoring,
	)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	conflictResult := query.First(&existingFolder)

	if conflictResult.RowsAffected > 0 {
		zap.L().Error("Folder name conflict detected",
			zap.String("folder_name", folder.Name))

		params.DB.Model(&folder).Update("status", models.FileStatusTrashed)
		return errors.New("folder name conflict")
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":     nil,
			"trashed_at": nil,
			"trashed_by": nil,
		}

		if err := tx.Model(&folder).Updates(updates).Error; err != nil {
			zap.L().Error("Failed to restore folder status", zap.Error(err))
			return err
		}

		// Unmark folder from storage
		objectPath := path.Join("folder", e.Payload.BucketID.String(), folder.ID.String())
		if err := params.Storage.UnmarkFileAsTrashed(objectPath); err != nil {
			zap.L().Warn("Failed to unmark folder as trashed",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("folder_id", e.Payload.FolderID.String()))
			// Continue - folders exist only in DB
		}

		// Restore child folders
		var childFolders []models.Folder
		if err := tx.Where(
			"bucket_id = ? AND folder_id = ? AND status = ? AND trashed_at = ?",
			e.Payload.BucketID,
			e.Payload.FolderID,
			models.FileStatusTrashed,
			folder.TrashedAt,
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

			if err := tx.Model(&models.Folder{}).
				Where("id IN ?", folderIDs).
				Updates(updates).Error; err != nil {
				zap.L().Error("Failed to restore child folders", zap.Error(err))
				return err
			}
		}

		// Restore child files
		var childFiles []models.File
		if err := tx.Where(
			"bucket_id = ? AND folder_id = ? AND status = ? AND trashed_at = ?",
			e.Payload.BucketID,
			e.Payload.FolderID,
			models.FileStatusTrashed,
			folder.TrashedAt,
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

				// Unmark each file from storage
				filePath := path.Join("buckets", e.Payload.BucketID.String(), child.ID.String())
				if err := params.Storage.UnmarkFileAsTrashed(filePath); err != nil {
					zap.L().Warn("Failed to unmark file as trashed",
						zap.Error(err),
						zap.String("file_id", child.ID.String()))
				}
			}

			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIDs).
				Updates(map[string]interface{}{
					"status":     models.FileStatusUploaded,
					"trashed_at": nil,
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

	// Check if there are remaining items to restore
	var remainingFolders int64
	params.DB.Model(&models.Folder{}).Where(
		"bucket_id = ? AND folder_id = ? AND status = ? AND trashed_at = ?",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusTrashed,
		folder.TrashedAt,
	).Count(&remainingFolders)

	var remainingFiles int64
	params.DB.Model(&models.File{}).Where(
		"bucket_id = ? AND folder_id = ? AND status = ? AND trashed_at = ?",
		e.Payload.BucketID,
		e.Payload.FolderID,
		models.FileStatusTrashed,
		folder.TrashedAt,
	).Count(&remainingFiles)

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to restore, requeuing event",
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to restore")
	}

	action := models.Activity{
		Message: activity.FolderRestored,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"folder_id":   e.Payload.FolderID.String(),
			"domain":      c.DefaultDomain,
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
