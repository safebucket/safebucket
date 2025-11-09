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

// parsedPathInfo contains the results of parsing an object path.
type parsedPathInfo struct {
	isMarker     bool
	originalPath string
	objectPath   string
	directory    string
	filename     string
}

// parseObjectPath parses the object key and extracts path information.
func (e *TrashExpiration) parseObjectPath(params *EventParams) parsedPathInfo {
	isMarker, originalPath := params.Storage.IsTrashMarkerPath(e.Payload.ObjectKey)
	prefix := path.Join("buckets", e.Payload.BucketID.String()) + "/"

	var objectPath string
	if isMarker {
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

	return parsedPathInfo{
		isMarker:     isMarker,
		originalPath: originalPath,
		objectPath:   objectPath,
		directory:    dir,
		filename:     filename,
	}
}

// findTrashedFile queries the database for a trashed file.
func (e *TrashExpiration) findTrashedFile(params *EventParams, pathInfo parsedPathInfo) (*models.File, error) {
	var file models.File
	result := params.DB.Where(
		"bucket_id = ? AND path = ? AND name = ? AND status = ?",
		e.Payload.BucketID,
		pathInfo.directory,
		pathInfo.filename,
		models.FileStatusTrashed,
	).First(&file)

	if result.Error != nil {
		zap.L().Warn("File not found in trash, skipping cleanup",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.String("path", pathInfo.directory),
			zap.String("name", pathInfo.filename),
			zap.Error(result.Error),
		)
		return nil, result.Error
	}

	return &file, nil
}

// handleFolderDeletion processes deletion of a folder and all its children.
func (e *TrashExpiration) handleFolderDeletion(params *EventParams, file *models.File, originalPath string) error {
	zap.L().Info("Processing folder marker expiration",
		zap.String("folder_id", file.ID.String()),
		zap.String("folder_path", originalPath))

	folderPath := path.Join(file.Path, file.Name)
	dbPath := fmt.Sprintf("%s/%%", folderPath)
	var childFiles []models.File

	// Single atomic query to fetch all children (direct and nested) to prevent race conditions
	if err := params.DB.Where(
		"bucket_id = ? AND (path = ? OR path LIKE ?)",
		e.Payload.BucketID,
		folderPath,
		dbPath,
	).Find(&childFiles).Error; err != nil {
		zap.L().Error("Failed to find children",
			zap.String("folder_id", file.ID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Info("Found children for folder deletion",
		zap.String("folder_id", file.ID.String()),
		zap.Int("total_children", len(childFiles)))

	if len(childFiles) > 0 {
		if err := e.deleteChildFiles(params, childFiles); err != nil {
			return err
		}
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

	return nil
}

// deleteChildFiles deletes child files from storage and database.
func (e *TrashExpiration) deleteChildFiles(params *EventParams, childFiles []models.File) error {
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
			zap.Error(err))
		return err
	}

	var childIDs []uuid.UUID
	for _, child := range childFiles {
		childIDs = append(childIDs, child.ID)
	}
	if err := params.DB.Where("id IN ?", childIDs).Delete(&models.File{}).Error; err != nil {
		zap.L().Error("Failed to delete child files from database",
			zap.Error(err))
		return err
	}

	zap.L().Info("Deleted child files",
		zap.Int("count", len(childFiles)))

	return nil
}

// handleFileDeletion processes deletion of a single file.
func (e *TrashExpiration) handleFileDeletion(params *EventParams, file *models.File, originalPath string) error {
	if err := params.Storage.RemoveObject(originalPath); err != nil {
		zap.L().Error("Failed to delete file from storage",
			zap.String("file_path", originalPath),
			zap.String("file_id", file.ID.String()),
			zap.Error(err),
		)
		if updateErr := params.DB.Model(file).Update("status", models.FileStatusTrashed).Error; updateErr != nil {
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

	return nil
}

func (e *TrashExpiration) callback(params *EventParams) error {
	zap.L().Info("Processing trash expiration event",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("object_key", e.Payload.ObjectKey),
	)

	pathInfo := e.parseObjectPath(params)

	file, err := e.findTrashedFile(params, pathInfo)
	if err != nil {
		return err
	}

	zap.L().Info("Processing file deletion from trash",
		zap.String("file_id", file.ID.String()),
		zap.String("file_name", file.Name),
		zap.Bool("is_marker_deletion", pathInfo.isMarker),
	)

	if pathInfo.isMarker && params.Storage != nil {
		if file.Type == models.FileTypeFolder {
			err = e.handleFolderDeletion(params, file, pathInfo.originalPath)
			if err != nil {
				return err
			}
		} else {
			err = e.handleFileDeletion(params, file, pathInfo.originalPath)
			if err != nil {
				return err
			}
		}
	}

	err = params.DB.Delete(file).Error
	if err != nil {
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
