package eventparser

import (
	"encoding/json"
	"strings"

	"github.com/safebucket/safebucket/internal/storage"

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
		return BucketEventTypeIgnore
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
		shareID := metadata["share_id"]

		if bucketID == "" || fileID == "" || (userID == "" && shareID == "") {
			zap.L().Warn("incomplete metadata in object",
				zap.String("object_key", record.S3.Object.Key),
				zap.String("bucket_id", bucketID),
				zap.String("file_id", fileID),
				zap.String("user_id", userID),
				zap.String("share_id", shareID))
			continue
		}

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			VersionID: metadata["version_id"],
			BucketID:  bucketID,
			FileID:    fileID,
			UserID:    userID,
			ShareID:   shareID,
		})
	}

	return uploadEvents
}
