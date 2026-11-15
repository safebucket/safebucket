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
		return BucketEventTypeIgnore
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
		shareID := event.Metadata["share-id"]

		if bucketID == "" || fileID == "" || (userID == "" && shareID == "") {
			zap.L().Warn("incomplete metadata in object",
				zap.String("bucket_id", bucketID),
				zap.String("file_id", fileID),
				zap.String("user_id", userID),
				zap.String("share_id", shareID))
			return uploadEvents
		}

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			VersionID: event.Metadata["version-id"],
			BucketID:  bucketID,
			FileID:    fileID,
			UserID:    userID,
			ShareID:   shareID,
		})
	} else {
		zap.L().Warn("event is not supported", zap.Any("event_type", msg.Metadata["eventType"]))
	}
	return uploadEvents
}
