package events

import (
	"path"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FolderPurgeName        = "FolderPurge"
	FolderPurgePayloadName = "FolderPurgePayload"
)

type FolderPurgePayload struct {
	Type     string
	BucketID uuid.UUID
	FolderID uuid.UUID
	UserID   uuid.UUID
}

func (p FolderPurgePayload) EventName() string { return p.Type }

type FolderPurge struct {
	Payload FolderPurgePayload
}

func NewFolderPurge(
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
) FolderPurge {
	return FolderPurge{
		Payload: FolderPurgePayload{
			Type:     FolderPurgeName,
			BucketID: bucketID,
			FolderID: folderID,
			UserID:   userID,
		},
	}
}

func (e *FolderPurge) callback(params *EventParams) error {
	folder, found, err := e.load(params)
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

	return e.finish(params, folder)
}

func (e *FolderPurge) load(params *EventParams) (models.Folder, bool, error) {
	var folder models.Folder
	result := params.DB.Unscoped().Where("id = ? AND bucket_id = ?",
		e.Payload.FolderID, e.Payload.BucketID).Find(&folder)
	if result.Error != nil {
		zap.L().Error("Failed to retrieve folder for deleting", zap.Error(result.Error))
		return folder, false, result.Error
	}
	if result.RowsAffected == 0 {
		zap.L().Info("Folder already purged, skipping",
			zap.String("folder_id", e.Payload.FolderID.String()))
		return folder, false, nil
	}
	if !folder.DeletedAt.Valid {
		zap.L().Warn("Folder not in trash, skipping delete",
			zap.String("folder_id", e.Payload.FolderID.String()))
		return folder, false, nil
	}
	return folder, true, nil
}

func (e *FolderPurge) deleteBatch(params *EventParams) (bool, error) {
	files, err := sql.FindTrashedFilesInTree(params.DB, e.Payload.BucketID, e.Payload.FolderID, c.BulkActionsLimit)
	if err != nil {
		zap.L().Error("Failed to find files to delete", zap.Error(err))
		return false, err
	}
	if len(files) > 0 {
		return true, e.deleteFiles(params, files)
	}

	deleted, err := sql.DeleteTrashedFoldersInTree(params.DB, e.Payload.BucketID, e.Payload.FolderID, c.BulkActionsLimit)
	if err != nil {
		zap.L().Error("Failed to delete subfolders", zap.Error(err))
		return false, err
	}
	return deleted > 0, nil
}

func (e *FolderPurge) deleteFiles(params *EventParams, files []models.File) error {
	_, err := sql.MarkFilesDeleting(params.DB, files...)
	if err != nil {
		zap.L().Error("Failed to mark files for deletion", zap.Error(err))
		return err
	}

	if err := params.Versions.DeleteFiles(files...); err != nil {
		zap.L().Error("Failed to delete files", zap.Error(err))
		return err
	}
	return nil
}

func (e *FolderPurge) finish(params *EventParams, folder models.Folder) error {
	var deleted bool
	err := params.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Where("deleted_at IS NOT NULL").Delete(&folder)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}

		deleted = true
		return outbox.AddActivities(tx, models.Activity{
			Message: activity.FolderDeleted,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   e.Payload.BucketID.String(),
				FolderID:   e.Payload.FolderID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		})
	})
	if err != nil {
		zap.L().Error("Failed to hard delete folder from database", zap.Error(err))
		return err
	}
	if !deleted {
		return nil
	}

	objectPath := path.Join("buckets", e.Payload.BucketID.String(), e.Payload.FolderID.String())
	if err = params.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
		zap.L().Warn("Failed to delete folder marker from storage",
			zap.Error(err),
			zap.String("folder_id", e.Payload.FolderID.String()))
	}

	zap.L().Info("Folder delete complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("folder_id", e.Payload.FolderID.String()),
	)
	return nil
}
