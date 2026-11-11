package events

import (
	"errors"
	"net/http"
	"path"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

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

func (p FolderRestorePayload) EventName() string { return p.Type }

type FolderRestore struct {
	Payload FolderRestorePayload
}

func NewFolderRestore(
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
) FolderRestore {
	return FolderRestore{
		Payload: FolderRestorePayload{
			Type:     FolderRestoreName,
			BucketID: bucketID,
			FolderID: folderID,
			UserID:   userID,
		},
	}
}

func (e *FolderRestore) callback(params *EventParams) error {
	var folders []models.Folder
	var files []models.File

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		root, found, err := e.lockRoot(tx)
		if err != nil || !found {
			return err
		}

		folders, err = sql.RestoreTreeFolders(tx, root.BucketID, root.ID, root.UpdatedAt, c.BulkActionsLimit)
		if err != nil {
			return err
		}

		remaining := c.BulkActionsLimit - len(folders)
		if remaining > 0 {
			files, err = sql.RestoreTreeFiles(tx, root.BucketID, root.ID, root.UpdatedAt, remaining)
			if err != nil {
				return err
			}
		}

		if len(folders)+len(files) == 0 {
			return e.finish(tx, root)
		}

		if err = outbox.AddActivities(tx, e.restoredActivities(folders, files)...); err != nil {
			return err
		}
		return outbox.AddEvents(tx, e.Payload)
	})
	if err != nil {
		zap.L().Error("Failed to restore folder content",
			zap.String("folder_id", e.Payload.FolderID.String()), zap.Error(err))
		return err
	}

	e.unmarkTrashed(params, folders, files)
	return nil
}

func (e *FolderRestore) lockRoot(tx *gorm.DB) (models.Folder, bool, error) {
	var root models.Folder
	if err := sql.LockBucketFileNames(tx, e.Payload.BucketID); err != nil {
		apiErr, ok := errors.AsType[*apierrors.APIError](err)

		if ok && apiErr.Status == http.StatusNotFound {
			zap.L().Info("Bucket deleted, skipping folder restore",
				zap.String("bucket_id", e.Payload.BucketID.String()))
			return root, false, nil
		}
		return root, false, err
	}

	result := tx.Unscoped().Where("id = ? AND bucket_id = ? AND status = ?",
		e.Payload.FolderID, e.Payload.BucketID, models.FolderStatusRestoring).Find(&root)

	if result.Error != nil {
		return root, false, result.Error
	}

	if result.RowsAffected == 0 {
		zap.L().Info("Folder is not being restored, stopping restore",
			zap.String("folder_id", e.Payload.FolderID.String()))
		return root, false, nil
	}
	return root, true, nil
}

func (e *FolderRestore) finish(tx *gorm.DB, root models.Folder) error {
	err := tx.Unscoped().Model(&root).Updates(map[string]any{
		"status":     models.FolderStatusCreated,
		"deleted_at": nil,
		"deleted_by": nil,
	}).Error
	if err == nil {
		zap.L().Info("Folder restore complete", zap.String("folder_id", root.ID.String()))
	}
	return err
}

func (e *FolderRestore) restoredActivities(folders []models.Folder, files []models.File) []models.Activity {
	activities := make([]models.Activity, 0, len(folders)+len(files))
	for _, folder := range folders {
		activities = append(activities, models.Activity{
			Message: activity.FolderRestored,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   folder.BucketID.String(),
				FolderID:   folder.ID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		})
	}

	for _, file := range files {
		activities = append(activities, models.Activity{
			Message: activity.FileRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		})
	}

	return activities
}

func (e *FolderRestore) unmarkTrashed(params *EventParams, folders []models.Folder, files []models.File) {
	for _, folder := range folders {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
		if err := params.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			zap.L().Warn("Failed to unmark folder as trashed",
				zap.Error(err), zap.String("folder_id", folder.ID.String()))
		}
	}

	for _, file := range files {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), file.ID.String())
		if err := params.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			zap.L().Warn("Failed to unmark file as trashed",
				zap.Error(err), zap.String("file_id", file.ID.String()))
		}
	}
}
