package events

import (
	"api/internal/models"
	"encoding/json"
	"path"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const TrashExpirationName = "TrashExpiration"
const TrashExpirationPayloadName = "TrashExpirationPayload"

type TrashExpirationPayload struct {
	Type      string    `json:"type"`
	BucketId  uuid.UUID `json:"bucket_id"`
	ObjectKey string    `json:"object_key"`
}

type TrashExpiration struct {
	Payload TrashExpirationPayload
}

// NewTrashExpirationFromBucketEvent creates a trash expiration event from a bucket deletion event
func NewTrashExpirationFromBucketEvent(bucketId uuid.UUID, objectKey string) *TrashExpiration {
	return &TrashExpiration{
		Payload: TrashExpirationPayload{
			Type:      TrashExpirationName,
			BucketId:  bucketId,
			ObjectKey: objectKey,
		},
	}
}

// Trigger publishes the trash expiration event (if needed for manual triggering)
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

// callback processes the trash expiration event when lifecycle policy deletes an object
func (e *TrashExpiration) callback(params *EventParams) error {
	zap.L().Info("Processing trash expiration event",
		zap.String("bucket_id", e.Payload.BucketId.String()),
		zap.String("object_key", e.Payload.ObjectKey),
	)

	objectPath := e.Payload.ObjectKey
	prefix := "buckets/" + e.Payload.BucketId.String() + "/"
	if len(objectPath) > len(prefix) {
		objectPath = objectPath[len(prefix):]
	}

	dir := path.Dir(objectPath)
	filename := path.Base(objectPath)

	zap.L().Debug("Parsed object path",
		zap.String("directory", dir),
		zap.String("filename", filename),
	)

	var file models.File
	result := params.DB.Where(
		"bucket_id = ? AND path = ? AND name = ? AND status = ?",
		e.Payload.BucketId,
		dir,
		filename,
		models.FileStatusTrashed,
	).First(&file)

	if result.Error != nil {
		zap.L().Warn("File not found in trash, skipping cleanup",
			zap.String("bucket_id", e.Payload.BucketId.String()),
			zap.String("path", dir),
			zap.String("name", filename),
			zap.Error(result.Error),
		)
		return nil
	}

	if file.TrashedAt != nil {
		daysSinceTrashed := time.Since(*file.TrashedAt).Hours() / 24
		retentionDays := float64(params.TrashRetentionDays)
		if daysSinceTrashed < retentionDays {
			zap.L().Error("Received expiration event for non-expired file",
				zap.String("file_id", file.ID.String()),
				zap.Float64("days_in_trash", daysSinceTrashed),
				zap.Float64("retention_days", retentionDays),
			)
			return nil
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
