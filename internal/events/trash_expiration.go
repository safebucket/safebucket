package events

import (
	"encoding/json"
	"path"
	"strings"

	"api/internal/activity"
	"api/internal/models"
	"api/internal/rbac"

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
	fileID, err := uuid.Parse(pathInfo.filename)
	if err != nil {
		zap.L().Warn("Invalid file ID in object path, skipping cleanup",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.String("filename", pathInfo.filename),
			zap.Error(err),
		)
		return nil, err
	}

	var files []models.File
	result := params.DB.Unscoped().Where(
		"bucket_id = ? AND id = ?",
		e.Payload.BucketID,
		fileID,
	).Find(&files)

	if result.Error != nil {
		zap.L().Warn("Database error while looking up file",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.String("file_id", fileID.String()),
			zap.Error(result.Error),
		)
		return nil, result.Error
	}

	if len(files) == 0 {
		return nil, nil
	}

	file := &files[0]

	if !file.DeletedAt.Valid {
		zap.L().Debug("File not soft-deleted, skipping expiration",
			zap.String("file_id", fileID.String()),
		)
		return nil, nil
	}

	return file, nil
}

// handleFileDeletion processes deletion of a single file.
func (e *TrashExpiration) handleFileDeletion(params *EventParams, file *models.File, originalPath string) error {
	if err := params.Storage.RemoveObject(originalPath); err != nil {
		zap.L().Error("Failed to delete file from storage",
			zap.String("file_path", originalPath),
			zap.String("file_id", file.ID.String()),
			zap.Error(err),
		)
		return err
	}
	zap.L().Info("Deleted file from storage",
		zap.String("file_path", originalPath),
		zap.String("file_id", file.ID.String()))

	return nil
}

func (e *TrashExpiration) callback(params *EventParams) error {
	zap.L().Debug("Processing trash expiration event",
		zap.String("bucket_id", e.Payload.BucketID.String()),
		zap.String("object_key", e.Payload.ObjectKey),
	)

	pathInfo := e.parseObjectPath(params)

	file, err := e.findTrashedFile(params, pathInfo)
	if err != nil {
		return err
	}

	if file == nil {
		zap.L().Debug("File not in trash, skipping expiration - likely already cleaned up",
			zap.String("bucket_id", e.Payload.BucketID.String()),
			zap.String("object_key", e.Payload.ObjectKey),
		)
		return nil
	}

	// At this point, file is guaranteed to be soft-deleted (deleted_at IS NOT NULL).
	//
	// However, there's a race condition: if a user restores the file,
	// UnmarkAsTrashed deletes the trash marker, which triggers this event.
	// The restore handler clears deleted_at, but this event may process
	// before or after that DB update completes.
	//
	// Solution: Re-check the file's deleted_at to distinguish:
	// - deleted_at IS NOT NULL → Lifecycle policy expiration, delete permanently
	// - deleted_at IS NULL → User restore in progress, skip deletion

	var currentFile models.File
	if reloadErr := params.DB.Unscoped().First(&currentFile, "id = ?", file.ID).Error; reloadErr != nil {
		zap.L().Error("Failed to reload file for status check",
			zap.String("file_id", file.ID.String()),
			zap.Error(reloadErr),
		)
		return reloadErr
	}

	if !currentFile.DeletedAt.Valid {
		zap.L().Info(
			"Trash marker deleted but file not soft-deleted - user restore in progress, skipping permanent deletion",
			zap.String("file_id", file.ID.String()),
		)
		return nil
	}

	zap.L().Info("Processing file deletion from trash",
		zap.String("file_id", file.ID.String()),
		zap.String("file_name", file.Name),
		zap.Bool("is_marker_deletion", pathInfo.isMarker),
	)

	if pathInfo.isMarker && params.Storage != nil {
		err = e.handleFileDeletion(params, file, pathInfo.originalPath)
		if err != nil {
			return err
		}
	}

	err = params.DB.Unscoped().Delete(file).Error
	if err != nil {
		zap.L().Error("Failed to hard delete file from database",
			zap.String("file_id", file.ID.String()),
			zap.Error(err),
		)
		return err
	}

	action := models.Activity{
		Message: activity.FileDeleted,
		Object:  file.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDelete.String(),
			"bucket_id":   e.Payload.BucketID.String(),
			"file_id":     file.ID.String(),
			"object_type": rbac.ResourceFile.String(),
		}),
	}

	if sendErr := params.ActivityLogger.Send(action); sendErr != nil {
		zap.L().Error("Failed to log trash expiration activity", zap.Error(sendErr))
	}

	zap.L().Info("Successfully processed trash expiration",
		zap.String("file_id", file.ID.String()),
		zap.String("name", file.Name),
	)

	return nil
}
