package eventparser

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type GCPEventParser struct{}

func (p *GCPEventParser) GetBucketEventType(msg *message.Message) string {
	eventType := msg.Metadata["eventType"]

	if eventType == "OBJECT_FINALIZE" {
		return BucketEventTypeUpload
	}

	if eventType == "OBJECT_DELETE" {
		return BucketEventTypeDeletion
	}

	return BucketEventTypeUnknown
}

func (p *GCPEventParser) ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent {
	var uploadEvents []BucketUploadEvent
	if msg.Metadata["eventType"] == "OBJECT_FINALIZE" {
		var event GCPEvent
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			zap.L().Error("event is unprocessable", zap.Error(err))
			return nil
		}

		bucketID := event.Metadata["bucket-id"]
		fileID := event.Metadata["file-id"]
		userID := event.Metadata["user-id"]

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
		})
	} else {
		zap.L().Warn("event is not supported", zap.Any("event_type", msg.Metadata["eventType"]))
	}
	return uploadEvents
}

func (p *GCPEventParser) ParseBucketDeletionEvents(
	msg *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	var deletionEvents []BucketDeletionEvent

	eventType := msg.Metadata["eventType"]
	if eventType == "OBJECT_DELETE" {
		objectKey := msg.Metadata["objectId"]
		if objectKey == "" {
			objectKey = msg.Metadata["name"]
		}

		if objectKey == "" {
			zap.L().Warn("deletion event missing object key",
				zap.Any("metadata", msg.Metadata))
			return nil
		}

		if bucketName := msg.Metadata["bucket"]; bucketName != "" && bucketName != expectedBucketName {
			zap.L().Debug("ignoring event from different bucket",
				zap.String("event_bucket", bucketName),
				zap.String("expected_bucket", expectedBucketName))
			return nil
		}

		bucketID := msg.Metadata["bucket-id"]

		if bucketID == "" {
			zap.L().Warn("unable to extract bucket ID from object key",
				zap.String("object_key", objectKey))
			return nil
		}

		deletionEvents = append(deletionEvents, BucketDeletionEvent{
			BucketID:  bucketID,
			ObjectKey: objectKey,
			EventName: eventType,
		})

		zap.L().Debug("parsed GCP deletion event",
			zap.String("event_type", eventType),
			zap.String("bucket_id", bucketID),
			zap.String("object_key", objectKey))
	}

	return deletionEvents
}
