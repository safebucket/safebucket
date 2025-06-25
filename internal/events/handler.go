package events

import (
	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/core"
	"api/internal/messaging"
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

func HandleBucketEvents(
	subscriber messaging.ISubscriber,
	db *gorm.DB,
	activityLogger activity.IActivityLogger,
	messages <-chan *message.Message,
) {
	for msg := range messages {
		zap.L().Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		uploadEvents := subscriber.ParseBucketUploadEvents(msg)

		for _, event := range uploadEvents {
			bucketUuid, err := uuid.Parse(event.BucketId)
			if err != nil {
				zap.L().Error("bucket id should be a valid UUID", zap.String("bucketId", event.BucketId))
				continue
			}

			fileUuid, err := uuid.Parse(event.FileId)
			if err != nil {
				zap.L().Error("file id should be a valid UUID", zap.String("fileId", event.FileId))
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
					"file_id":     event.FileId,
					"bucket_id":   event.BucketId,
					"user_id":     event.UserId,
				}),
			}

			err = activityLogger.Send(action)

			if err != nil {
				zap.L().Error("failed to send activity", zap.Error(err))
			}
		}

		msg.Ack()
	}
}
