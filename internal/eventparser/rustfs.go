package eventparser

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

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

	if eventName == "s3:ObjectCreated:Post" || eventName == "s3:ObjectCreated:Put" ||
		eventName == "s3:ObjectCreated:CompleteMultipartUpload" {
		return BucketEventTypeUpload
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
		shareID := metadata["share-id"]

		if bucketID == "" || fileID == "" || (userID == "" && shareID == "") {
			zap.L().Warn("incomplete metadata in object",
				zap.String("bucket_id", bucketID),
				zap.String("file_id", fileID),
				zap.String("user_id", userID),
				zap.String("share_id", shareID))
			continue
		}

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			VersionID: metadata["version-id"],
			BucketID:  bucketID,
			FileID:    fileID,
			UserID:    userID,
			ShareID:   shareID,
		})
	}

	return uploadEvents
}
