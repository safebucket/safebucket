package eventparser

import (
	"encoding/json"
	"strings"

	"api/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type AWSEventParser struct {
	Storage storage.IStorage
}

func (p *AWSEventParser) GetBucketEventType(msg *message.Message) string {
	var event AWSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(event.Records) == 0 {
		return BucketEventTypeUnknown
	}

	eventName := event.Records[0].EventName

	if strings.HasPrefix(eventName, "ObjectCreated:") {
		return BucketEventTypeUpload
	}

	if strings.HasPrefix(eventName, "ObjectRemoved:") ||
		strings.HasPrefix(eventName, "LifecycleExpiration:") {
		return BucketEventTypeDeletion
	}

	zap.L().Warn("Unrecognized S3 event type",
		zap.String("eventName", eventName))
	return BucketEventTypeUnknown
}

func (p *AWSEventParser) ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent {
	var event AWSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		return nil
	}

	if p.Storage == nil {
		zap.L().Error("storage is not initialized for AWS event parser")
		return nil
	}

	var uploadEvents []BucketUploadEvent
	for _, record := range event.Records {
		metadata, err := p.Storage.StatObject(record.S3.Object.Key)
		if err != nil {
			zap.L().Error("failed to stat object",
				zap.String("object_key", record.S3.Object.Key),
				zap.Error(err))
			continue
		}

		bucketID := metadata["bucket_id"]
		fileID := metadata["file_id"]
		userID := metadata["user_id"]

		if bucketID == "" || fileID == "" || userID == "" {
			zap.L().Warn("incomplete metadata in object",
				zap.String("object_key", record.S3.Object.Key),
				zap.String("bucket_id", bucketID),
				zap.String("file_id", fileID),
				zap.String("user_id", userID))
			continue
		}

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
		})
	}

	return uploadEvents
}

func (p *AWSEventParser) ParseBucketDeletionEvents(
	msg *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	var event AWSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		if record.S3.Bucket.Name != expectedBucketName {
			zap.L().Debug("ignoring event from different bucket",
				zap.String("event_bucket", record.S3.Bucket.Name),
				zap.String("expected_bucket", expectedBucketName))
			continue
		}

		objectKey := record.S3.Object.Key
		bucketID := ExtractBucketID(objectKey)
		if bucketID == "" {
			zap.L().Warn("unable to extract bucket ID from object key",
				zap.String("object_key", objectKey),
				zap.String("event_name", record.EventName))
			continue
		}

		deletionEvents = append(deletionEvents, BucketDeletionEvent{
			BucketID:  bucketID,
			ObjectKey: objectKey,
			EventName: record.EventName,
		})

		zap.L().Debug("parsed deletion event",
			zap.String("event_name", record.EventName),
			zap.String("bucket_id", bucketID),
			zap.String("object_key", objectKey))
	}

	return deletionEvents
}
