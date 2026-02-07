package eventparser

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type MinIOEventParser struct{}

func (p *MinIOEventParser) GetBucketEventType(msg *message.Message) string {
	var event MinIOEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(event.Records) == 0 {
		return BucketEventTypeUnknown
	}

	eventName := event.Records[0].EventName
	objectKey := event.Records[0].S3.Object.Key

	decodedKey, err := url.QueryUnescape(objectKey)
	if err != nil {
		zap.L().Debug("Failed to URL decode object key, using raw key",
			zap.String("raw_key", objectKey),
			zap.Error(err))
		decodedKey = objectKey
	}

	if eventName == "s3:ObjectCreated:Post" || eventName == "s3:ObjectCreated:Put" {
		if strings.HasPrefix(decodedKey, "trash/") {
			zap.L().Debug("Ignoring trash marker creation event",
				zap.String("event_name", eventName),
				zap.String("object_key", decodedKey))
			return BucketEventTypeIgnore
		}
		return BucketEventTypeUpload
	}

	if strings.HasPrefix(eventName, "s3:ObjectRemoved:") ||
		strings.HasPrefix(eventName, "s3:LifecycleExpiration:") {
		return BucketEventTypeDeletion
	}

	zap.L().Debug("Unrecognized S3 event type",
		zap.String("event_name", eventName),
		zap.String("raw_payload", string(msg.Payload)))

	return BucketEventTypeIgnore
}

func (p *MinIOEventParser) ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent {
	var event MinIOEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		return nil
	}

	var uploadEvents []BucketUploadEvent
	for _, record := range event.Records {
		metadata := record.S3.Object.UserMetadata

		bucketID := metadata["X-Amz-Meta-Bucket-Id"]
		fileID := metadata["X-Amz-Meta-File-Id"]
		userID := metadata["X-Amz-Meta-User-Id"]

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
		})
	}

	return uploadEvents
}

func (p *MinIOEventParser) ParseBucketDeletionEvents(
	msg *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	var event MinIOEvent
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

		objectKey, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil {
			zap.L().Warn("failed to URL decode object key",
				zap.String("raw_key", record.S3.Object.Key),
				zap.Error(err))
			objectKey = record.S3.Object.Key
		}

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
