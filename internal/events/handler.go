package events

import (
	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/core"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/sql"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"reflect"
)

type Event interface {
	callback(webUrl string, mailer *core.Mailer)
}

func getEventFromMessage(eventType string, msg *message.Message) (Event, error) {
	payloadType, exists := eventRegistry[fmt.Sprintf("%sPayload", eventType)]

	if !exists {
		return nil, fmt.Errorf("payload type %s not found in event registry", eventType)
	}

	payload := reflect.New(payloadType).Interface()

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message payload: %w", err)
	}

	eventTyp, exists := eventRegistry[eventType]
	if !exists {
		return nil, fmt.Errorf("event type %s not found in event registry", eventType)
	}

	eventInstance := reflect.New(eventTyp).Interface()

	eventValue := reflect.ValueOf(eventInstance).Elem()
	payloadField := eventValue.FieldByName("Payload")
	if !payloadField.IsValid() || !payloadField.CanSet() {
		return nil, fmt.Errorf("event type %s does not have a settable 'Payload' field", eventType)
	}
	payloadField.Set(reflect.ValueOf(payload).Elem())

	event, ok := eventInstance.(Event)
	if !ok {
		return nil, fmt.Errorf("type %s does not implement Event interface", eventType)
	}

	return event, nil
}

func HandleNotifications(webUrl string, mailer *core.Mailer, messages <-chan *message.Message) {
	for msg := range messages {
		zap.L().Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		eventType := msg.Metadata.Get("type")
		event, err := getEventFromMessage(eventType, msg)

		if err != nil {
			zap.L().Error("event is misconfigured", zap.Error(err))
			msg.Ack()
			continue
		}

		event.callback(webUrl, mailer)
		msg.Ack()
	}
}

func HandleBucketEvents(db *gorm.DB, activityLogger activity.IActivityLogger, messages <-chan *message.Message) {
	for msg := range messages {
		zap.L().Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		var event S3Event
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			zap.L().Error("event is unprocessable", zap.Error(err))
			msg.Ack()
			continue
		}

		for _, record := range event.Records {
			if record.EventName == "s3:ObjectCreated:Post" {
				bucketId := record.S3.Object.UserMetadata["X-Amz-Meta-Bucket-Id"]
				fileId := record.S3.Object.UserMetadata["X-Amz-Meta-File-Id"]
				userId := record.S3.Object.UserMetadata["X-Amz-Meta-User-Id"]

				bucketUuid, err := uuid.Parse(bucketId)
				if err != nil {
					zap.L().Error("bucket id should be a valid UUID", zap.String("bucketId", bucketId))
					continue
				}

				fileUuid, err := uuid.Parse(fileId)
				if err != nil {
					zap.L().Error("file id should be a valid UUID", zap.String("fileId", fileId))
					continue
				}

				file, err := sql.GetFileById(db, bucketUuid, fileUuid)
				if err != nil {
					zap.L().Error("event is misconfigured", zap.Error(err))
					continue
				}

				db.Model(&file).Update("uploaded", true)

				action := models.Activity{
					Message: activity.FileUploaded,
					Filter: activity.NewLogFilter(map[string]string{
						"action":      rbac.ActionCreate.String(),
						"domain":      c.DefaultDomain,
						"object_type": rbac.ResourceBucket.String(),
						"file_id":     fileId,
						"bucket_id":   bucketId,
						"user_id":     userId,
					}),
				}

				err = activityLogger.Send(action)

				if err != nil {
					zap.L().Error("failed to send activity", zap.Error(err))
				}
			} else {
				zap.L().Warn("event is not supported", zap.Any("event_name", record.EventName))
				continue
			}
		}

		msg.Ack()
	}
}
