package events

import (
	"encoding/json"
	"errors"
	"path"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ItemsTrashName        = "ItemsTrash"
	ItemsTrashPayloadName = "ItemsTrashPayload"
)

type ItemsTrashPayload struct {
	Type      string
	BucketID  uuid.UUID
	FolderIDs []uuid.UUID
	FileIDs   []uuid.UUID
	UserID    uuid.UUID
}

type ItemsTrash struct {
	Publisher messaging.IPublisher
	Payload   ItemsTrashPayload
}

func NewItemsTrash(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	folderIDs []uuid.UUID,
	fileIDs []uuid.UUID,
	userID uuid.UUID,
) ItemsTrash {
	return ItemsTrash{
		Publisher: publisher,
		Payload: ItemsTrashPayload{
			Type:      ItemsTrashName,
			BucketID:  bucketID,
			FolderIDs: folderIDs,
			FileIDs:   fileIDs,
			UserID:    userID,
		},
	}
}

func (e *ItemsTrash) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling items trash event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger items trash event", zap.Error(err))
	}
}

//nolint:gocognit // Complex event handler logic with multiple validation steps
func (e *ItemsTrash) callback(params *EventParams) error {
	zap.L().Info("Starting bulk trash",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.Int("folders", len(e.Payload.FolderIDs)),
		zap.Int("files", len(e.Payload.FileIDs)),
	)

	var folders []models.Folder
	var files []models.File

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		if len(e.Payload.FolderIDs) > 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("bucket_id = ? AND id IN ? AND status != ?",
					e.Payload.BucketID, e.Payload.FolderIDs, models.FolderStatusRestoring).
				Limit(c.BulkActionsLimit).
				Find(&folders).Error; err != nil {
				zap.L().Error("Failed to lock folders for bulk trash", zap.Error(err))
				return err
			}
		}

		if len(e.Payload.FileIDs) > 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("bucket_id = ? AND id IN ? AND status != ?",
					e.Payload.BucketID, e.Payload.FileIDs, models.FileStatusRestoring).
				Limit(c.BulkActionsLimit).
				Find(&files).Error; err != nil {
				zap.L().Error("Failed to lock files for bulk trash", zap.Error(err))
				return err
			}
		}

		for _, folder := range folders {
			objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
			if err := params.Storage.MarkAsTrashed(objectPath, folder); err != nil {
				zap.L().Warn("Failed to mark folder as trashed in storage",
					zap.Error(err),
					zap.String("folder_id", folder.ID.String()))
			}
		}

		for _, file := range files {
			objectPath := path.Join("buckets", e.Payload.BucketID.String(), file.ID.String())
			if err := params.Storage.MarkAsTrashed(objectPath, file); err != nil {
				zap.L().Warn("Failed to mark file as trashed in storage",
					zap.Error(err),
					zap.String("file_id", file.ID.String()))
			}
		}

		if len(folders) > 0 {
			folderUpdates := map[string]interface{}{
				"status":     models.FolderStatusDeleted,
				"deleted_by": e.Payload.UserID,
			}
			if err := tx.Model(&models.Folder{}).
				Where("id IN ?", folderIDList(folders)).
				Updates(folderUpdates).Error; err != nil {
				zap.L().Error("Failed to update folders for bulk trash", zap.Error(err))
				return err
			}

			if err := tx.Where("id IN ?", folderIDList(folders)).Delete(&models.Folder{}).Error; err != nil {
				zap.L().Error("Failed to soft delete folders for bulk trash", zap.Error(err))
				return err
			}
		}

		if len(files) > 0 {
			fileUpdates := map[string]interface{}{
				"status":     models.FileStatusDeleted,
				"deleted_by": e.Payload.UserID,
			}
			if err := tx.Model(&models.File{}).
				Where("id IN ?", fileIDList(files)).
				Updates(fileUpdates).Error; err != nil {
				zap.L().Error("Failed to update files for bulk trash", zap.Error(err))
				return err
			}

			if err := tx.Where("id IN ?", fileIDList(files)).Delete(&models.File{}).Error; err != nil {
				zap.L().Error("Failed to soft delete files for bulk trash", zap.Error(err))
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if err = params.ActivityLogger.Send(models.Activity{
			Message: activity.FolderTrashed,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionErase.String(),
				BucketID:   e.Payload.BucketID.String(),
				FolderID:   folder.ID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		}); err != nil {
			zap.L().Warn("Failed to log trash activity",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
		}

		e.triggerChildTrash(params, folder.ID)
	}

	for _, file := range files {
		if err = params.ActivityLogger.Send(models.Activity{
			Message: activity.FileTrashed,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionErase.String(),
				BucketID:   e.Payload.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		}); err != nil {
			zap.L().Warn("Failed to log trash activity",
				zap.Error(err),
				zap.String("file_id", file.ID.String()))
		}
	}

	var remainingFolders int64
	if len(e.Payload.FolderIDs) > 0 {
		params.DB.Model(&models.Folder{}).Where(
			"bucket_id = ? AND id IN ? AND status != ?",
			e.Payload.BucketID,
			e.Payload.FolderIDs,
			models.FolderStatusRestoring,
		).Count(&remainingFolders)
	}

	var remainingFiles int64
	if len(e.Payload.FileIDs) > 0 {
		params.DB.Model(&models.File{}).Where(
			"bucket_id = ? AND id IN ? AND status != ?",
			e.Payload.BucketID,
			e.Payload.FileIDs,
			models.FileStatusRestoring,
		).Count(&remainingFiles)
	}

	if remainingFolders > 0 || remainingFiles > 0 {
		zap.L().Info("More items to trash, requeuing event",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.Int64("remaining_folders", remainingFolders),
			zap.Int64("remaining_files", remainingFiles))
		return errors.New("remaining items to trash")
	}

	zap.L().Info("Bulk trash complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.Int("trashed_folders", len(folders)),
		zap.Int("trashed_files", len(files)),
	)

	return nil
}

// triggerChildTrash publishes an ItemsTrash event for the direct children of a
// trashed folder so subtree processing continues asynchronously.
func (e *ItemsTrash) triggerChildTrash(params *EventParams, folderID uuid.UUID) {
	var childFolders []models.Folder
	if err := params.DB.Where("bucket_id = ? AND folder_id = ?",
		e.Payload.BucketID, folderID).Find(&childFolders).Error; err != nil {
		zap.L().Error("Failed to find child folders for bulk trash",
			zap.Error(err),
			zap.String("folder_id", folderID.String()))
		return
	}

	var childFiles []models.File
	if err := params.DB.Where("bucket_id = ? AND folder_id = ?",
		e.Payload.BucketID, folderID).Find(&childFiles).Error; err != nil {
		zap.L().Error("Failed to find child files for bulk trash",
			zap.Error(err),
			zap.String("folder_id", folderID.String()))
		return
	}

	if len(childFolders) == 0 && len(childFiles) == 0 {
		return
	}

	childEvent := NewItemsTrash(
		params.Publisher,
		e.Payload.BucketID,
		folderIDList(childFolders),
		fileIDList(childFiles),
		e.Payload.UserID,
	)
	childEvent.Trigger()
}

func folderIDList(folders []models.Folder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(folders))
	for _, folder := range folders {
		ids = append(ids, folder.ID)
	}
	return ids
}

func fileIDList(files []models.File) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	return ids
}
