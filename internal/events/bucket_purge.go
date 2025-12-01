package events

import (
	"encoding/json"
	"errors"
	"path"
	"time"

	c "api/internal/configuration"
	"api/internal/messaging"
	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
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

type BucketPurge struct {
	Publisher messaging.IPublisher
	Payload   BucketPurgePayload
}

func NewBucketPurge(
	publisher messaging.IPublisher,
	bucketID uuid.UUID,
	userID uuid.UUID,
) BucketPurge {
	return BucketPurge{
		Publisher: publisher,
		Payload: BucketPurgePayload{
			Type:     BucketPurgeName,
			BucketID: bucketID,
			UserID:   userID,
		},
	}
}

func (e *BucketPurge) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling bucket purge event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger bucket purge event", zap.Error(err))
	}
}

func (e *BucketPurge) callback(params *EventParams) error {
	zap.L().Info("Starting bucket purge (permanent deletion)",
		zap.String("bucket_id", e.Payload.BucketID.String()),
	)

	// Phase 1: Delete root-level files
	if !e.deleteRootFiles(params) {
		return errors.New("remaining files to delete")
	}

	// Phase 2: Delete root-level folders (delegate to FolderPurge)
	if !e.deleteRootFolders(params) {
		return errors.New("remaining folders to delete")
	}

	// Phase 3: Cleanup orphaned storage objects
	if !e.cleanupOrphanedStorage(params) {
		return errors.New("remaining storage objects")
	}

	zap.L().Info("Bucket purge complete",
		zap.String("bucket_id", e.Payload.BucketID.String()),
	)

	return nil
}

// deleteRootFiles deletes all root-level files (folder_id IS NULL) for the bucket.
func (e *BucketPurge) deleteRootFiles(params *EventParams) bool {
	// Query root-level files only (not in any folder)
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

	// Delete in transaction
	err := params.DB.Transaction(func(tx *gorm.DB) error {
		// Build storage paths and collect file IDs
		var storagePaths []string
		var fileIDs []uuid.UUID
		for _, file := range files {
			fileIDs = append(fileIDs, file.ID)
			filePath := path.Join("buckets", e.Payload.BucketID.String(), file.ID.String())
			storagePaths = append(storagePaths, filePath)
		}

		// Delete from storage first
		if len(storagePaths) > 0 {
			if err := params.Storage.RemoveObjects(storagePaths); err != nil {
				zap.L().Warn("Failed to delete files from storage", zap.Error(err))
				// Continue - files may not exist in storage yet (uploading status)
			} else {
				zap.L().Info("Successfully deleted files from storage",
					zap.Int("count", len(storagePaths)),
				)
			}
		}

		// Hard delete from database (Unscoped removes soft-deleted items permanently)
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

	// Check if more files exist (batching)
	var remainingCount int64
	params.DB.Unscoped().Model(&models.File{}).
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More root files to delete, requeuing",
			zap.Int64("remaining", remainingCount),
		)
		return false // Requeue event
	}

	return true
}

// deleteRootFolders delegates deletion of root-level folders to FolderPurge events.
func (e *BucketPurge) deleteRootFolders(params *EventParams) bool {
	// Query root-level folders only (not in any parent folder)
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

	// Trigger FolderPurge event for each root folder
	for _, folder := range folders {
		// Ensure folder is in trashed status (FolderPurge requirement)
		// Use Unscoped to update soft-deleted records
		params.DB.Unscoped().Model(&folder).Updates(map[string]interface{}{
			"status":     models.FileStatusTrashed,
			"deleted_at": time.Now(),
			"deleted_by": e.Payload.UserID,
		})

		// Trigger FolderPurge event
		purgeEvent := NewFolderPurge(
			params.Publisher,
			folder.BucketID,
			folder.ID,
			e.Payload.UserID,
		)
		purgeEvent.Trigger()

		zap.L().Debug("Triggered FolderPurge event",
			zap.String("folder_id", folder.ID.String()),
			zap.String("folder_name", folder.Name),
		)
	}

	// Check if more folders exist (batching)
	var remainingCount int64
	params.DB.Unscoped().Model(&models.Folder{}).
		Where("bucket_id = ? AND folder_id IS NULL", e.Payload.BucketID).
		Count(&remainingCount)

	if remainingCount > 0 {
		zap.L().Info("More root folders to delete, requeuing",
			zap.Int64("remaining", remainingCount),
		)
		return false // Requeue event
	}

	return true
}

// cleanupOrphanedStorage removes any remaining objects in storage that weren't in the database.
// This handles edge cases like uncommitted uploads, residual storage artifacts, and trash markers.
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

	if err := params.Storage.RemoveObjects(objects); err != nil {
		zap.L().Error("Failed to delete orphaned storage objects", zap.Error(err))
		return false
	}

	zap.L().Info("Successfully cleaned up orphaned storage objects",
		zap.Int("count", len(objects)),
	)

	// Check if more objects exist (batching)
	if len(objects) == c.BulkActionsLimit {
		zap.L().Info("More orphaned objects may exist, requeuing")
		return false
	}

	return true
}
