package events

import (
	"encoding/json"
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
	Type     string
	BucketID uuid.UUID
	UserID   uuid.UUID
}

type ItemsTrash struct {
	Publisher messaging.IPublisher
	Payload   ItemsTrashPayload
}

func NewItemsTrash(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	userID uuid.UUID,
) ItemsTrash {
	return ItemsTrash{
		Publisher: publisher,
		Payload: ItemsTrashPayload{
			Type:     ItemsTrashName,
			BucketID: bucketID,
			UserID:   userID,
		},
	}
}

func (e *ItemsTrash) Trigger() {
	if err := e.publish(); err != nil {
		zap.L().Error("Failed to trigger items trash event", zap.Error(err))
	}
}

func (e *ItemsTrash) publish() error {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	return e.Publisher.Publish(msg)
}

func (e *ItemsTrash) callback(params *EventParams) error {
	var files []models.File
	var folders []models.Folder

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		if err := e.lockDeletingItems(tx, &files, &folders); err != nil {
			return err
		}
		if err := e.addTrashMarker(params, files, folders); err != nil {
			return err
		}
		return e.deleteItems(tx, files, folders)
	})
	if err != nil {
		return err
	}

	e.logActivities(params, files, folders)

	processed := len(files) + len(folders)
	if processed == c.TrashBatchLimit {
		next := NewItemsTrash(params.Publisher, e.Payload.BucketID, e.Payload.UserID)
		if err = next.publish(); err != nil {
			return err
		}

		zap.L().Info("Queued next trash batch",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.Int("processed", processed))
		return nil
	}

	zap.L().Info("Bulk trash complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("user_id", e.Payload.UserID.String()))

	return nil
}

func (e *ItemsTrash) lockDeletingItems(tx *gorm.DB, files *[]models.File, folders *[]models.Folder) error {
	query := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("bucket_id = ? AND status = ? AND deleted_by = ?",
			e.Payload.BucketID, models.FileStatusDeleting, e.Payload.UserID).
		Limit(c.TrashBatchLimit)
	if err := query.Find(files).Error; err != nil {
		return err
	}

	remainingSlots := c.TrashBatchLimit - len(*files)
	if remainingSlots <= 0 {
		return nil
	}

	query = tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("bucket_id = ? AND status = ? AND deleted_by = ?",
			e.Payload.BucketID, models.FolderStatusDeleting, e.Payload.UserID).
		Limit(remainingSlots)
	return query.Find(folders).Error
}

func (e *ItemsTrash) addTrashMarker(params *EventParams, files []models.File, folders []models.Folder) error {
	for _, file := range files {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), file.ID.String())
		if err := params.Storage.MarkAsTrashed(objectPath, file); err != nil {
			return err
		}
	}

	for _, folder := range folders {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
		if err := params.Storage.MarkAsTrashed(objectPath, folder); err != nil {
			return err
		}
	}

	return nil
}

func (e *ItemsTrash) deleteItems(tx *gorm.DB, files []models.File, folders []models.Folder) error {
	if len(files) > 0 {
		ids := fileIDs(files)
		if err := tx.Model(&models.File{}).Where("id IN ?", ids).
			Update("status", models.FileStatusDeleted).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.File{}).Error; err != nil {
			return err
		}
	}

	if len(folders) > 0 {
		ids := folderIDs(folders)
		if err := tx.Model(&models.Folder{}).Where("id IN ?", ids).
			Update("status", models.FolderStatusDeleted).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.Folder{}).Error; err != nil {
			return err
		}
	}

	return nil
}

// TODO(YLB): Future improvement possible here, send multiple activity at once.
func (e *ItemsTrash) logActivities(params *EventParams, files []models.File, folders []models.Folder) {
	for _, file := range files {
		if err := params.ActivityLogger.Send(fileTrashedActivity(file, e.Payload.UserID)); err != nil {
			zap.L().Error("Failed to log file trash activity",
				zap.String("file_id", file.ID.String()),
				zap.Error(err))
		}
	}

	for _, folder := range folders {
		if err := params.ActivityLogger.Send(folderTrashedActivity(folder, e.Payload.UserID)); err != nil {
			zap.L().Error("Failed to log folder trash activity",
				zap.String("folder_id", folder.ID.String()),
				zap.Error(err))
		}
	}
}

func fileTrashedActivity(file models.File, userID uuid.UUID) models.Activity {
	return models.Activity{
		Message: activity.FileTrashed,
		Object:  file.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     rbac.ActionErase.String(),
			BucketID:   file.BucketID.String(),
			FileID:     file.ID.String(),
			ObjectType: rbac.ResourceFile.String(),
			UserID:     userID.String(),
		}),
	}
}

func folderTrashedActivity(folder models.Folder, userID uuid.UUID) models.Activity {
	return models.Activity{
		Message: activity.FolderTrashed,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     rbac.ActionErase.String(),
			BucketID:   folder.BucketID.String(),
			FolderID:   folder.ID.String(),
			ObjectType: rbac.ResourceFolder.String(),
			UserID:     userID.String(),
		}),
	}
}

func fileIDs(files []models.File) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	return ids
}

func folderIDs(folders []models.Folder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(folders))
	for _, folder := range folders {
		ids = append(ids, folder.ID)
	}
	return ids
}
