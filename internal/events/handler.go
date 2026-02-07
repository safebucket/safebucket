package events

import (
	"encoding/json"
	"fmt"
	"reflect"

	"api/internal/activity"
	"api/internal/eventparser"
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/notifier"
	"api/internal/rbac"
	"api/internal/sql"
	"api/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EventParams struct {
	WebURL             string
	Notifier           notifier.INotifier
	Publisher          messaging.IPublisher
	DB                 *gorm.DB
	Storage            storage.IStorage
	ActivityLogger     activity.IActivityLogger
	TrashRetentionDays int
}

type Event interface {
	callback(params *EventParams) error
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

func HandleEvents(params *EventParams, messages <-chan *message.Message) {
	for msg := range messages {
		zap.L().
			Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		eventType := msg.Metadata.Get("type")
		event, err := getEventFromMessage(eventType, msg)
		if err != nil {
			zap.L().Error("event is misconfigured", zap.Error(err))
			msg.Ack()
			continue
		}

		if err = event.callback(params); err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	}
}

func HandleBucketEvents(
	parser eventparser.IBucketEventParser,
	db *gorm.DB,
	activityLogger activity.IActivityLogger,
	storage storage.IStorage,
	trashRetentionDays int,
	messages <-chan *message.Message,
) {
	for msg := range messages {
		zap.L().
			Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		eventType := parser.GetBucketEventType(msg)

		switch eventType {
		case eventparser.BucketEventTypeUpload:
			uploadEvents := parser.ParseBucketUploadEvents(msg)

			for _, event := range uploadEvents {
				bucketUUID, err := uuid.Parse(event.BucketID)
				if err != nil {
					zap.L().
						Error("bucket id should be a valid UUID", zap.String("bucketId", event.BucketID))
					continue
				}

				fileUUID, err := uuid.Parse(event.FileID)
				if err != nil {
					zap.L().Error("file id should be a valid UUID", zap.String("fileID", event.FileID))
					continue
				}

				file, err := sql.GetFileByID(db, bucketUUID, fileUUID)
				if err != nil {
					zap.L().Error("event is misconfigured", zap.Error(err))
					continue
				}

				db.Model(&file).Update("status", models.FileStatusUploaded)

				action := models.Activity{
					Message: activity.FileUploaded,
					Object:  file.ToActivity(),
					Filter: activity.NewLogFilter(map[string]string{
						"action":      rbac.ActionCreate.String(),
						"object_type": rbac.ResourceFile.String(),
						"file_id":     event.FileID,
						"bucket_id":   event.BucketID,
						"user_id":     event.UserID,
					}),
				}

				err = activityLogger.Send(action)
				if err != nil {
					zap.L().Error("failed to send activity", zap.Error(err))
				}
			}

		case eventparser.BucketEventTypeDeletion:
			deletionEvents := parser.ParseBucketDeletionEvents(msg, storage.GetBucketName())

			for _, event := range deletionEvents {
				bucketUUID, err := uuid.Parse(event.BucketID)
				if err != nil {
					zap.L().
						Error("bucket id should be a valid UUID", zap.String("bucketId", event.BucketID))
					continue
				}
				trashEvent := NewTrashExpirationFromBucketEvent(bucketUUID, event.ObjectKey)

				params := &EventParams{
					DB:                 db,
					Storage:            storage,
					ActivityLogger:     activityLogger,
					TrashRetentionDays: trashRetentionDays,
				}

				if err = trashEvent.callback(params); err != nil {
					zap.L().Error("Failed to process trash expiration", zap.Error(err))
				}
			}

		case eventparser.BucketEventTypeIgnore:
			zap.L().Debug("ignoring event", zap.String("raw_payload", string(msg.Payload)))

		default:
			zap.L().Warn("Unknown bucket event type", zap.String("payload", string(msg.Payload)))
		}

		msg.Ack()
	}
}
