package events

import (
	"path"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"

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
	BucketID uuid.UUID
	UserID   uuid.UUID
}

type ItemsTrash struct {
	Payload ItemsTrashPayload
}

func (e *ItemsTrash) callback(params *EventParams) error {
	var files []models.File
	var folders []models.Folder

	if err := e.selectDeletingItems(params.DB, &files, &folders); err != nil {
		return err
	}

	if err := e.addTrashMarker(params, files, folders); err != nil {
		return err
	}

	var finalizedFiles []models.File
	var finalizedFolders []models.Folder
	err := params.DB.Transaction(func(tx *gorm.DB) error {
		if txErr := e.lockSelectedDeletingItems(tx, files, folders, &finalizedFiles, &finalizedFolders); txErr != nil {
			return txErr
		}
		if txErr := e.deleteItems(tx, finalizedFiles, finalizedFolders); txErr != nil {
			return txErr
		}
		if txErr := e.enqueueActivities(tx, finalizedFiles, finalizedFolders); txErr != nil {
			return txErr
		}

		if len(finalizedFiles)+len(finalizedFolders) == c.BatchLimit {
			return outbox.EnqueueEvent(tx, ItemsTrashName, e.Payload)
		}
		return nil
	})
	if err != nil {
		return err
	}

	processed := len(finalizedFiles) + len(finalizedFolders)
	if processed == c.BatchLimit {
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

func (e *ItemsTrash) lockSelectedDeletingItems(
	tx *gorm.DB,
	files []models.File,
	folders []models.Folder,
	lockedFiles *[]models.File,
	lockedFolders *[]models.Folder,
) error {
	if len(files) > 0 {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ? AND status = ? AND deleted_by = ?", fileIDs(files), models.FileStatusDeleting, e.Payload.UserID).
			Find(lockedFiles).Error; err != nil {
			return err
		}
	}

	if len(folders) > 0 {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ? AND status = ? AND deleted_by = ?", folderIDs(folders), models.FolderStatusDeleting, e.Payload.UserID).
			Find(lockedFolders).Error
	}

	return nil
}

func (e *ItemsTrash) selectDeletingItems(db *gorm.DB, files *[]models.File, folders *[]models.Folder) error {
	query := db.Where("bucket_id = ? AND status = ? AND deleted_by = ?",
		e.Payload.BucketID, models.FileStatusDeleting, e.Payload.UserID).
		Limit(c.BatchLimit)
	if err := query.Find(files).Error; err != nil {
		return err
	}

	remainingSlots := c.BatchLimit - len(*files)
	if remainingSlots <= 0 {
		return nil
	}

	query = db.Where("bucket_id = ? AND status = ? AND deleted_by = ?",
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

func (e *ItemsTrash) enqueueActivities(tx *gorm.DB, files []models.File, folders []models.Folder) error {
	for _, file := range files {
		if err := outbox.EnqueueActivity(tx, fileTrashedActivity(file, e.Payload.UserID)); err != nil {
			return err
		}
	}

	for _, folder := range folders {
		if err := outbox.EnqueueActivity(tx, folderTrashedActivity(folder, e.Payload.UserID)); err != nil {
			return err
		}
	}

	return nil
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
