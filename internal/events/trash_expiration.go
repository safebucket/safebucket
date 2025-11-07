package events

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	TrashExpirationName        = "TrashExpiration"
	TrashExpirationPayloadName = "TrashExpirationPayload"
)

type TrashExpirationPayload struct {
	Type      string    `json:"type"`
	BucketID  uuid.UUID `json:"bucket_id"`
	ObjectKey string    `json:"object_key"`
}

type TrashExpiration struct {
	Payload TrashExpirationPayload
}

// NewTrashExpirationFromBucketEvent creates a trash expiration event from a bucket deletion event.
func NewTrashExpirationFromBucketEvent(bucketID uuid.UUID, objectKey string) *TrashExpiration {
	return &TrashExpiration{
		Payload: TrashExpirationPayload{
			Type:      TrashExpirationName,
			BucketID:  bucketID,
			ObjectKey: objectKey,
		},
	}
}

// Trigger publishes the trash expiration event (if needed for manual triggering).
func (e *TrashExpiration) Trigger(publisher message.Publisher) {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling trash expiration event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = publisher.Publish("events", msg)
	if err != nil {
		zap.L().Error("failed to trigger trash expiration event", zap.Error(err))
	}
}

func (e *TrashExpiration) callback(params *EventParams) error {
	zap.L().Info("Processing trash expiration event",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("object_key", e.Payload.ObjectKey),
	)

	isMarker, originalPath := params.Storage.IsTrashMarkerPath(e.Payload.ObjectKey) //TODO: Review @yoh

	var objectPath string
	if isMarker {
		prefix := "buckets/" + e.Payload.BucketID.String() + "/"
		if len(originalPath) > len(prefix) {
			objectPath = originalPath[len(prefix):]
		} else {
			objectPath = originalPath
		}
		zap.L().Info("Detected trash marker deletion",
			zap.String("marker_path", e.Payload.ObjectKey),
			zap.String("original_path", originalPath),
			zap.String("relative_path", objectPath))
	} else {
		objectPath = e.Payload.ObjectKey
		prefix := "buckets/" + e.Payload.BucketID.String() + "/"
		if len(objectPath) > len(prefix) {
			objectPath = objectPath[len(prefix):]
		}
	}

	dir := path.Dir(objectPath)
	filename := path.Base(objectPath)

	if dir == "." {
		dir = "/"
	} else if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}

	zap.L().Debug("Parsed object path",
		zap.String("directory", dir),
		zap.String("filename", filename),
		zap.Bool("is_marker", isMarker),
	)

	var file models.File
	result := params.DB.Where(
		"bucket_id = ? AND path = ? AND name = ? AND status = ?",
		e.Payload.BucketID,
		dir,
		filename,
		models.FileStatusTrashed,
	).First(&file)

	if result.Error != nil {
		zap.L().Warn("File not found in trash, skipping cleanup",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.String("path", dir),
			zap.String("name", filename),
			zap.Error(result.Error),
		)
		return result.Error
	}

	zap.L().Info("Processing file deletion from trash",
		zap.String("file_id", file.ID.String()),
		zap.String("file_name", file.Name),
		zap.Bool("is_marker_deletion", isMarker),
	)

	if isMarker && params.Storage != nil {
		if file.Type == models.FileTypeFolder {
			zap.L().Info("Processing folder marker expiration",
				zap.String("folder_id", file.ID.String()),
				zap.String("folder_path", originalPath))

			folderPath := path.Join(file.Path, file.Name)
			var childFiles []models.File

			if err := params.DB.Where(
				"bucket_id = ? AND path = ?",
				e.Payload.BucketID,
				folderPath,
			).Find(&childFiles).Error; err != nil {
				zap.L().Error("Failed to find direct children",
					zap.String("folder_id", file.ID.String()),
					zap.Error(err))
				return err
			}

			dbPath := fmt.Sprintf("%s/%%", folderPath)
			var nestedFiles []models.File
			if err := params.DB.Where(
				"bucket_id = ? AND path LIKE ?",
				e.Payload.BucketID,
				dbPath,
			).Find(&nestedFiles).Error; err != nil {
				zap.L().Error("Failed to find nested children",
					zap.String("folder_id", file.ID.String()),
					zap.Error(err))
				return err
			}

			childFiles = append(childFiles, nestedFiles...)

			zap.L().Info("Found children for folder deletion",
				zap.String("folder_id", file.ID.String()),
				zap.Int("total_children", len(childFiles)))

			if len(childFiles) > 0 {
				var storagePaths []string
				for _, child := range childFiles {
					childPath := path.Join(
						"buckets",
						e.Payload.BucketID.String(),
						child.Path,
						child.Name,
					)
					storagePaths = append(storagePaths, childPath)
				}

				if err := params.Storage.RemoveObjects(storagePaths); err != nil {
					zap.L().Error("Failed to delete child files from storage",
						zap.String("folder_id", file.ID.String()),
						zap.Error(err))
					return err
				}

				var childIDs []uuid.UUID
				for _, child := range childFiles {
					childIDs = append(childIDs, child.ID)
				}
				if err := params.DB.Where("id IN ?", childIDs).Delete(&models.File{}).Error; err != nil {
					zap.L().Error("Failed to delete child files from database",
						zap.String("folder_id", file.ID.String()),
						zap.Error(err))
					return err
				}

				zap.L().Info("Deleted child files",
					zap.String("folder_id", file.ID.String()),
					zap.Int("count", len(childFiles)))
			}

			if err := params.Storage.RemoveObject(originalPath); err != nil {
				zap.L().Error("Failed to delete folder from storage",
					zap.String("folder_path", originalPath),
					zap.String("folder_id", file.ID.String()),
					zap.Error(err))
				return err
			}

			zap.L().Info("Deleted folder from storage",
				zap.String("folder_path", originalPath),
				zap.String("folder_id", file.ID.String()))
		} else {
			if err := params.Storage.RemoveObject(originalPath); err != nil {
				zap.L().Error("Failed to delete file from storage",
					zap.String("file_path", originalPath),
					zap.String("file_id", file.ID.String()),
					zap.Error(err),
				)
				if updateErr := params.DB.Model(&file).Update("status", models.FileStatusTrashed).Error; updateErr != nil {
					zap.L().Error("Failed to revert file status",
						zap.String("file_id", file.ID.String()),
						zap.Error(updateErr),
					)
				}
				return err
			}
			zap.L().Info("Deleted file from storage",
				zap.String("file_path", originalPath),
				zap.String("file_id", file.ID.String()))
		}
	}

	if err := params.DB.Delete(&file).Error; err != nil {
		zap.L().Error("Failed to soft delete file from database",
			zap.String("file_id", file.ID.String()),
			zap.Error(err),
		)
		return err
	}

	zap.L().Info("Successfully processed trash expiration",
		zap.String("file_id", file.ID.String()),
		zap.String("name", file.Name),
	)

	return nil
}
