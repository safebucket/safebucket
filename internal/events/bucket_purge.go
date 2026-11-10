package events

import (
	"path"

	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
	found, err := e.load(params)
	if err != nil || !found {
		return err
	}

	more, err := e.deleteBatch(params)
	if err != nil {
		return err
	}
	if more {
		return outbox.AddEvents(params.DB, e.Payload)
	}

	zap.L().Info("Bucket deletion complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
	)
	return nil
}

func (e *BucketPurge) load(params *EventParams) (bool, error) {
	var bucket models.Bucket
	result := params.DB.Unscoped().Where("id = ?", e.Payload.BucketID).Find(&bucket)
	if result.Error != nil {
		zap.L().Error("Failed to retrieve bucket for deleting", zap.Error(result.Error))
		return false, result.Error
	}
	if result.RowsAffected == 0 || !bucket.DeletedAt.Valid {
		zap.L().Warn("Bucket not deleted, skipping deletion",
			zap.String("bucket_id", e.Payload.BucketID.String()))
		return false, nil
	}
	return true, nil
}

func (e *BucketPurge) deleteBatch(params *EventParams) (bool, error) {
	if more, err := e.deleteFiles(params); err != nil || more {
		return more, err
	}
	if more, err := e.deleteFolders(params); err != nil || more {
		return more, err
	}
	return e.deleteObjects(params)
}

func (e *BucketPurge) deleteFiles(params *EventParams) (bool, error) {
	var files []models.File
	result := params.DB.Unscoped().
		Where("bucket_id = ?", e.Payload.BucketID).
		Limit(c.BulkActionsLimit).
		Find(&files)

	if result.Error != nil {
		zap.L().Error("Failed to query files", zap.Error(result.Error))
		return false, result.Error
	}

	if len(files) == 0 {
		return false, nil
	}

	_, err := sql.MarkFilesDeleting(params.DB, files...)
	if err != nil {
		zap.L().Error("Failed to mark files for deletion", zap.Error(err))
		return false, err
	}

	if err = params.Versions.DeleteFiles(files...); err != nil {
		zap.L().Error("Failed to delete files", zap.Error(err))
		return false, err
	}
	return true, nil
}

func (e *BucketPurge) deleteFolders(params *EventParams) (bool, error) {
	var ids []uuid.UUID
	result := params.DB.Unscoped().Model(&models.Folder{}).
		Where("bucket_id = ?", e.Payload.BucketID).
		Limit(c.BulkActionsLimit).
		Pluck("id", &ids)

	if result.Error != nil {
		zap.L().Error("Failed to query folders", zap.Error(result.Error))
		return false, result.Error
	}

	if len(ids) == 0 {
		return false, nil
	}

	if err := params.DB.Unscoped().Where("id IN ?", ids).Delete(&models.Folder{}).Error; err != nil {
		zap.L().Error("Failed to delete folders", zap.Error(err))
		return false, err
	}
	return true, nil
}

func (e *BucketPurge) deleteObjects(params *EventParams) (bool, error) {
	prefixes := []string{
		path.Join("buckets", e.Payload.BucketID.String()),
		path.Join("trash", e.Payload.BucketID.String()),
	}
	for _, prefix := range prefixes {
		objects, err := params.Storage.ListObjects(prefix, c.BulkActionsLimit)
		if err != nil {
			zap.L().Error("Failed to list storage objects for cleanup",
				zap.Error(err), zap.String("prefix", prefix))
			return false, err
		}
		if len(objects) == 0 {
			continue
		}

		if err = params.Storage.RemoveObjects(objects); err != nil {
			zap.L().Error("Failed to delete storage objects",
				zap.Error(err), zap.String("prefix", prefix))
			return false, err
		}
		return true, nil
	}
	return false, nil
}
