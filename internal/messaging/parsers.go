package messaging

import (
	"encoding/json"
	"net/url"
	"strings"

	"api/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

// NewBucketEventParser returns the appropriate parser for the given storage provider type.
func NewBucketEventParser(storageType string, store storage.IStorage) IBucketEventParser {
	switch storageType {
	case "rustfs":
		return &RustFSEventParser{}
	case "minio":
		return &MinIOEventParser{}
	case "aws":
		return &AWSEventParser{Storage: store}
	case "gcp":
		return &GCPEventParser{}
	case "s3":
		return &MinIOEventParser{}
	default:
		return &RustFSEventParser{}
	}
}

type RustFSEventParser struct{}

func (p *RustFSEventParser) GetBucketEventType(msg *message.Message) string {
	var event RustFSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(event.Records) == 0 {
		return BucketEventTypeUnknown
	}

	eventName := event.Records[0].EventName
	objectKey := event.Records[0].Data.S3.Object.Key

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

func (p *RustFSEventParser) ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent {
	var event RustFSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		return nil
	}

	var uploadEvents []BucketUploadEvent
	for _, record := range event.Records {
		metadata := record.Data.S3.Object.UserMetadata

		bucketID := metadata["bucket-id"]
		fileID := metadata["file-id"]
		userID := metadata["user-id"]

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
		})
	}

	return uploadEvents
}

func (p *RustFSEventParser) ParseBucketDeletionEvents(
	msg *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	var event RustFSEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		if record.Data.S3.Bucket.Name != expectedBucketName {
			zap.L().Debug("ignoring event from different bucket",
				zap.String("event_bucket", record.Data.S3.Bucket.Name),
				zap.String("expected_bucket", expectedBucketName))
			continue
		}

		objectKey, err := url.QueryUnescape(record.Data.S3.Object.Key)
		if err != nil {
			zap.L().Warn("failed to URL decode object key",
				zap.String("raw_key", record.Data.S3.Object.Key),
				zap.Error(err))
			objectKey = record.Data.S3.Object.Key
		}

		zap.L().Debug("received deletion/expiration event",
			zap.String("event_name", record.EventName),
			zap.String("object_key", objectKey),
			zap.String("raw_payload", string(msg.Payload)),
			zap.Any("user_metadata", record.Data.S3.Object.UserMetadata),
			zap.String("bucket_name", record.Data.S3.Bucket.Name),
			zap.Int64("size", record.Data.S3.Object.Size))

		bucketID := extractBucketID(objectKey)
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

		bucketID := extractBucketID(objectKey)
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

// --- AWS ---

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
		bucketID := extractBucketID(objectKey)
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

func extractBucketID(objectKey string) string {
	if strings.HasPrefix(objectKey, "buckets/") || strings.HasPrefix(objectKey, "trash/") {
		parts := strings.Split(objectKey, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}
