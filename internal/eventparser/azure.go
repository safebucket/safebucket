package eventparser

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/safebucket/safebucket/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const (
	azureBlobCreatedEvent = "Microsoft.Storage.BlobCreated"
	azureBlobDeletedEvent = "Microsoft.Storage.BlobDeleted"
)

type AzureEventParser struct {
	Storage storage.IStorage
}

func (p *AzureEventParser) GetBucketEventType(msg *message.Message) string {
	events, err := parseAzureBlobEvents(msg.Payload)
	if err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(events) == 0 {
		return BucketEventTypeUnknown
	}

	event := events[0]
	switch azureEventType(event) {
	case azureBlobCreatedEvent:
		if strings.HasPrefix(azureObjectKey(event), "trash/") {
			zap.L().Debug("Ignoring trash marker creation event", zap.String("subject", event.Subject))
			return BucketEventTypeIgnore
		}
		return BucketEventTypeUpload
	case azureBlobDeletedEvent:
		return BucketEventTypeDeletion
	default:
		zap.L().Debug("Unrecognized Azure event type",
			zap.String("event_type", azureEventType(event)),
			zap.String("raw_payload", string(msg.Payload)))
		return BucketEventTypeIgnore
	}
}

func (p *AzureEventParser) ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent {
	events, err := parseAzureBlobEvents(msg.Payload)
	if err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		return nil
	}

	var uploadEvents []BucketUploadEvent
	for _, event := range events {
		if azureEventType(event) != azureBlobCreatedEvent {
			continue
		}

		objectKey := azureObjectKey(event)
		if strings.HasPrefix(objectKey, "trash/") {
			continue
		}

		metadata, statErr := p.Storage.StatObject(objectKey)
		if statErr != nil {
			zap.L().Error("failed to stat object", zap.String("object_key", objectKey), zap.Error(statErr))
			continue
		}

		bucketID := metadata["bucket_id"]
		fileID := metadata["file_id"]
		userID := metadata["user_id"]
		shareID := metadata["share_id"]

		if bucketID == "" || fileID == "" || (userID == "" && shareID == "") {
			zap.L().Warn("incomplete metadata in object",
				zap.String("object_key", objectKey),
				zap.String("bucket_id", bucketID),
				zap.String("file_id", fileID),
				zap.String("user_id", userID),
				zap.String("share_id", shareID))
			continue
		}

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
			ShareID:  shareID,
		})
	}

	return uploadEvents
}

func (p *AzureEventParser) ParseBucketDeletionEvents(
	msg *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	events, err := parseAzureBlobEvents(msg.Payload)
	if err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, event := range events {
		if azureEventType(event) != azureBlobDeletedEvent {
			continue
		}

		container := azureContainerFromSubject(event.Subject)
		if container != "" && container != expectedBucketName {
			zap.L().Debug("ignoring event from different container",
				zap.String("event_container", container),
				zap.String("expected_container", expectedBucketName))
			continue
		}

		objectKey := azureObjectKey(event)
		bucketID := ExtractBucketID(objectKey)
		if bucketID == "" {
			zap.L().Warn("unable to extract bucket ID from object key",
				zap.String("object_key", objectKey),
				zap.String("event_type", azureEventType(event)))
			continue
		}

		deletionEvents = append(deletionEvents, BucketDeletionEvent{
			BucketID:  bucketID,
			ObjectKey: objectKey,
			EventName: azureEventType(event),
		})
	}

	return deletionEvents
}

func parseAzureBlobEvents(payload []byte) ([]AzureBlobEvent, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var events []AzureBlobEvent
		if err := json.Unmarshal(trimmed, &events); err != nil {
			return nil, err
		}
		return events, nil
	}

	var event AzureBlobEvent
	if err := json.Unmarshal(trimmed, &event); err != nil {
		return nil, err
	}

	return []AzureBlobEvent{event}, nil
}

func azureEventType(event AzureBlobEvent) string {
	if event.EventType != "" {
		return event.EventType
	}
	return event.Type
}

func azureObjectKey(event AzureBlobEvent) string {
	key := azureKeyFromSubject(event.Subject)
	if key == "" {
		key = azureKeyFromURL(event.Data.URL)
	}

	decoded, err := url.QueryUnescape(key)
	if err != nil {
		return key
	}
	return decoded
}

func azureKeyFromSubject(subject string) string {
	const marker = "/blobs/"
	if idx := strings.Index(subject, marker); idx != -1 {
		return subject[idx+len(marker):]
	}
	return ""
}

func azureKeyFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	trimmedPath := strings.TrimPrefix(parsed.Path, "/")
	parts := strings.SplitN(trimmedPath, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func azureContainerFromSubject(subject string) string {
	const marker = "/containers/"
	idx := strings.Index(subject, marker)
	if idx == -1 {
		return ""
	}

	rest := subject[idx+len(marker):]
	if end := strings.Index(rest, "/"); end != -1 {
		return rest[:end]
	}
	return rest
}
