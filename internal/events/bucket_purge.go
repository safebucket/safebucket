package events

import (
	"errors"
	"path"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	BucketPurgeName        = "BucketPurge"
	BucketPurgePayloadName = "BucketPurgePayload"
)

type BucketPurgePayload struct {
	Type     string
	BucketID uuid.UUID
	UserID   uuid.UUID
}

func (p BucketPurgePayload) EventName() string { return p.Type }

type BucketPurge struct {
	Payload BucketPurgePayload
}

func NewBucketPurge(
	bucketID uuid.UUID,
	userID uuid.UUID,
) BucketPurge {
	return BucketPurge{
		Payload: BucketPurgePayload{
			Type:     BucketPurgeName,
			BucketID: bucketID,
			UserID:   userID,
		},
	}
}

func (e *BucketPurge) callback(params *EventParams) error {
	zap.L().Info("Starting bucket purge (permanent deletion)",
		zap.String("bucket_id", e.Payload.BucketID.String()),
	)

	if !e.deleteRootFiles(params) {
		return errors.New("remaining files to delete")
	}

	if !e.deleteRootFolders(params) {
		return errors.New("remaining folders to delete")
	}

	if !e.cleanupOrphanedStorage(params) {
		return errors.New("remaining storage objects")
	}

	zap.L().Info("Bucket purge complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
	)

	return nil
}

func (e *BucketPurge) deleteRootFiles(params *EventParams) bool {
	var files []models.File
	result := params.DB.Unscoped().
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Order("created_at ASC").
		Limit(c.BulkActionsLimit).
		Find(&files)

	if result.Error != nil {
		zap.L().Error("Failed to query root files", zap.Error(result.Error))
		return false
	}

	if len(files) == 0 {
		zap.L().Info("No root-level files to delete")
		return true
	}

	zap.L().Info("Processing root-level files for deletion",
		zap.Int("count", len(files)),
	)

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		if err := params.Versions.RemoveFileObjects(tx, files...); err != nil {
			return err
		}
		var fileIDs []uuid.UUID
		for _, file := range files {
			fileIDs = append(fileIDs, file.ID)
		}

		if err := tx.Unscoped().Where("id IN ?", fileIDs).Delete(&models.File{}).Error; err != nil {
			zap.L().Error("Failed to delete files from database", zap.Error(err))
			return err
		}

		zap.L().Info("Successfully deleted root files from database",
			zap.Int("count", len(files)),
		)

		return nil
	})

	if err != nil {
		zap.L().Error("Transaction failed for root file deletion", zap.Error(err))
		return false
	}

	var remainingCount int64
	params.DB.Unscoped().Model(&models.File{}).
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More root files to delete, requeuing",
			zap.Int64("remaining", remainingCount),
		)
		return false
	}

	return true
}

func (e *BucketPurge) deleteRootFolders(params *EventParams) bool {
	var folders []models.Folder
	result := params.DB.Unscoped().
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Order("created_at ASC").
		Limit(c.BulkActionsLimit).
		Find(&folders)

	if result.Error != nil {
		zap.L().Error("Failed to query root folders", zap.Error(result.Error))
		return false
	}

	if len(folders) == 0 {
		zap.L().Info("No root-level folders to delete")
		return true
	}

	zap.L().Info("Triggering FolderPurge for root folders",
		zap.Int("count", len(folders)),
	)

	payloads := make([]FolderPurgePayload, 0, len(folders))
	for _, folder := range folders {
		payloads = append(payloads, NewFolderPurge(folder.BucketID, folder.ID, e.Payload.UserID).Payload)
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Model(&models.Folder{}).
			Where("id IN ?", folderIDs(folders)).
			Updates(map[string]interface{}{
				"deleted_at": time.Now(),
				"deleted_by": e.Payload.UserID,
			}).Error; err != nil {
			return err
		}
		return outbox.AddEvents(tx, payloads...)
	})
	if err != nil {
		zap.L().Error("Failed to queue FolderPurge for root folders", zap.Error(err))
		return false
	}

	var remainingCount int64
	params.DB.Unscoped().Model(&models.Folder{}).
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More root folders to delete, requeuing",
			zap.Int64("remaining", remainingCount),
		)
		return false
	}

	return true
}

func (e *BucketPurge) cleanupOrphanedStorage(params *EventParams) bool {
	bucketPrefix := path.Join("buckets", e.Payload.BucketID.String())

	objects, err := params.Storage.ListObjects(bucketPrefix, c.BulkActionsLimit)
	if err != nil {
		zap.L().Error("Failed to list storage objects for cleanup", zap.Error(err))
		return false
	}

	if len(objects) == 0 {
		zap.L().Info("No orphaned storage objects found")
		return true
	}

	zap.L().Info("Cleaning up orphaned storage objects",
		zap.Int("count", len(objects)),
	)

	if err = params.Storage.RemoveObjects(objects); err != nil {
		zap.L().Error("Failed to delete orphaned storage objects", zap.Error(err))
		return false
	}

	zap.L().Info("Successfully cleaned up orphaned storage objects",
		zap.Int("count", len(objects)),
	)

	if len(objects) == c.BulkActionsLimit {
		zap.L().Info("More orphaned objects may exist, requeuing")
		return false
	}

	return true
}
