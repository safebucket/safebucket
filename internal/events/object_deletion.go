package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"

	c "api/internal/configuration"
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/storage"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	ObjectDeletionName        = "ObjectDeletion"
	ObjectDeletionPayloadName = "ObjectDeletionPayload"
)

type ObjectDeletionPayload struct {
	Type   string
	Bucket models.Bucket
	Path   string
}

type ObjectDeletion struct {
	Publisher messaging.IPublisher
	Payload   ObjectDeletionPayload
}

func NewObjectDeletion(
	publisher messaging.IPublisher,
	bucket models.Bucket,
	path string,
) ObjectDeletion {
	return ObjectDeletion{
		Publisher: publisher,
		Payload: ObjectDeletionPayload{
			Type:   ObjectDeletionName,
			Bucket: bucket,
			Path:   path,
		},
	}
}

func (e *ObjectDeletion) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger event", zap.Error(err))
	}
}

func (e *ObjectDeletion) callback(params *EventParams) error {
	zap.L().Info("Starting object deletion",
		zap.String("bucket_id", e.Payload.Bucket.ID.String()),
		zap.String("path", e.Payload.Path),
	)

	var files []models.File
	objectsPath := path.Join("buckets", e.Payload.Bucket.ID.String(), e.Payload.Path)

	dbPath := fmt.Sprintf("%s%%", e.Payload.Path)
	zap.L().Info("Getting batch of files to delete",
		zap.String("bucket_id", e.Payload.Bucket.ID.String()),
		zap.String("path", dbPath),
	)
	result := params.DB.Where(
		"bucket_id = ? AND path LIKE ?", e.Payload.Bucket.ID, dbPath).
		Limit(c.BulkActionsLimit).
		Find(&files)

	if result.Error != nil {
		zap.L().Error("Failed to get files for deletion", zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		zap.L().Info("No files found in database, checking storage for orphaned files")
		done := e.cleanupOrphanedFiles(params.Storage, objectsPath)
		if !done {
			zap.L().Info("Remaining files... requeuing deletion")
			return errors.New("remaining files")
		}
		return nil
	}

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		zap.L().Info("Deleting files from database", zap.Int("count", len(files)))

		var fileIDs []uuid.UUID
		for _, file := range files {
			fileIDs = append(fileIDs, file.ID)
		}
		dbResult := tx.Where("id IN ?", fileIDs).Delete(&models.File{})
		if dbResult.Error != nil {
			zap.L().Error("Failed to delete files from database", zap.Error(dbResult.Error))
			return dbResult.Error
		}
		zap.L().
			Info("Successfully deleted files from database", zap.Int64("count", dbResult.RowsAffected))

		zap.L().Info("Deleting files from storage", zap.Int("count", len(files)))
		var storagePaths []string
		for _, file := range files {
			storagePaths = append(
				storagePaths,
				path.Join("buckets", file.BucketID.String(), file.Path, file.Name),
			)
		}

		if len(storagePaths) > 0 {
			err := params.Storage.RemoveObjects(storagePaths)
			if err != nil {
				zap.L().Error("Failed to delete files from storage", zap.Error(err))
			}
		}

		zap.L().Info("Successfully deleted files from storage", zap.Int("count", len(storagePaths)))

		return nil
	})
	if err != nil {
		return err
	}

	zap.L().Info("Checking if we need to delete more files")

	if len(files) == c.BulkActionsLimit {
		result = params.DB.Where(
			"bucket_id = ? AND path LIKE ?", e.Payload.Bucket.ID, dbPath).
			Find(&files)

		if result.RowsAffected > 0 {
			zap.L().
				Info("More files to delete, requeuing event", zap.Int64("count", result.RowsAffected))
			return errors.New("remaining files left")
		}
	}

	zap.L().Info("File deletion complete, performing final cleanup")

	done := e.cleanupOrphanedFiles(params.Storage, objectsPath)
	if !done {
		return errors.New("remaining files left")
	}

	if e.Payload.Path != "/" {
		parentPath := path.Dir(e.Payload.Path)
		folderName := path.Base(e.Payload.Path)

		result = params.DB.Where("bucket_id = ? AND name = ? AND path = ? AND type = 'folder'",
			e.Payload.Bucket.ID, folderName, parentPath).Delete(&models.File{})

		if result.Error != nil {
			zap.L().Error("Failed to delete folder from database", zap.Error(result.Error))
			return result.Error
		}

		if result.RowsAffected > 0 {
			zap.L().Info("Successfully deleted folder",
				zap.String("folder_name", folderName),
				zap.String("parent_path", parentPath),
			)
		}
	}

	zap.L().Info("Object deletion complete",
		zap.String("bucket_id", e.Payload.Bucket.ID.String()),
		zap.String("path", e.Payload.Path),
	)

	return nil
}

// cleanupOrphanedFiles performs a final check for any files left in storage that weren't in the database.
func (e *ObjectDeletion) cleanupOrphanedFiles(storage storage.IStorage, bucketPrefix string) bool {
	objects, err := storage.ListObjects(bucketPrefix, c.BulkActionsLimit)
	if err != nil {
		zap.L().Error("Failed to list objects for orphan cleanup", zap.Error(err))
		return true
	}

	if len(objects) == 0 {
		zap.L().Info("No orphaned files found in storage")
		return true
	}

	err = storage.RemoveObjects(objects)
	if err != nil {
		zap.L().Error("Failed to delete orphaned files", zap.Error(err))
		return false
	}

	zap.L().Info("Successfully cleaned up orphaned files", zap.Int("count", len(objects)))

	// If we found the maximum number of objects, there might be more - requeue for another cleanup
	if len(objects) == c.BulkActionsLimit {
		zap.L().Info("More orphaned files may exist, requeuing cleanup")
		return false
	}
	return true
}
