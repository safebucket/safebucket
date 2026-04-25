package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/eventparser"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

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
	Cache              cache.ICache
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

func HandleEvents(ctx context.Context, params *EventParams, messages <-chan *message.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
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
}

func handleUploadEvents(
	parser eventparser.IBucketEventParser,
	msg *message.Message,
	db *gorm.DB,
	activityLogger activity.IActivityLogger,
	publisher messaging.IPublisher,
) {
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
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				ObjectType: rbac.ResourceFile.String(),
				FileID:     event.FileID,
				BucketID:   event.BucketID,
				UserID:     event.UserID,
			}),
		}

		err = activityLogger.Send(action)
		if err != nil {
			zap.L().Error("failed to send activity", zap.Error(err))
		}

		var bucket models.Bucket
		if err = db.Where("id = ?", bucketUUID).First(&bucket).Error; err != nil {
			continue
		}

		if event.ShareID != "" {
			shareUUID, parseErr := uuid.Parse(event.ShareID)
			if parseErr != nil {
				zap.L().Error("share id should be a valid UUID", zap.String("shareID", event.ShareID))
				continue
			}
			if err = activityLogger.Send(models.Activity{
				Message: activity.ShareFileUploaded,
				Object:  file.ToActivity(),
				Filter: activity.NewLogFilter(models.ActivityFields{
					Action:     rbac.ActionCreate.String(),
					ObjectType: rbac.ResourceFile.String(),
					FileID:     event.FileID,
					BucketID:   event.BucketID,
					ShareID:    shareUUID.String(),
				}),
			}); err != nil {
				zap.L().Error("failed to send activity", zap.Error(err))
			}

			var share models.Share
			if err = db.Where("id = ?", shareUUID).First(&share).Error; err != nil {
				continue
			}

			var user models.User
			if err = db.Where("id = ?", share.CreatedBy).First(&user).Error; err != nil {
				continue
			}

			evt := NewFileActivityNotification(
				publisher, FileActivityUpload, FileActivitySourceShare,
				bucketUUID, bucket.Name, file.Name, share.CreatedBy, user.Email,
			)
			evt.Trigger()
		} else {
			userUUID, parseErr := uuid.Parse(event.UserID)
			if parseErr != nil {
				zap.L().Error("user id should be a valid UUID", zap.String("userID", event.UserID))
				continue
			}

			if err = activityLogger.Send(models.Activity{
				Message: activity.FileUploaded,
				Object:  file.ToActivity(),
				Filter: activity.NewLogFilter(models.ActivityFields{
					Action:     rbac.ActionCreate.String(),
					ObjectType: rbac.ResourceFile.String(),
					FileID:     event.FileID,
					BucketID:   event.BucketID,
					UserID:     event.UserID,
				}),
			}); err != nil {
				zap.L().Error("failed to send activity", zap.Error(err))
			}

			var user models.User
			if err = db.Where("id = ?", userUUID).First(&user).Error; err != nil {
				continue
			}

			evt := NewFileActivityNotification(
				publisher, FileActivityUpload, FileActivitySourceUser,
				bucketUUID, bucket.Name, file.Name, userUUID, user.Email,
			)
			evt.Trigger()
		}
	}
}

func handleDeletionEvents(
	parser eventparser.IBucketEventParser,
	msg *message.Message,
	db *gorm.DB,
	storage storage.IStorage,
	activityLogger activity.IActivityLogger,
	trashRetentionDays int,
) {
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
}

func HandleBucketEvents(
	ctx context.Context,
	parser eventparser.IBucketEventParser,
	db *gorm.DB,
	activityLogger activity.IActivityLogger,
	storage storage.IStorage,
	publisher messaging.IPublisher,
	trashRetentionDays int,
	messages <-chan *message.Message,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			zap.L().
				Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

			eventType := parser.GetBucketEventType(msg)

			switch eventType {
			case eventparser.BucketEventTypeUpload:
				handleUploadEvents(parser, msg, db, activityLogger, publisher)

			case eventparser.BucketEventTypeDeletion:
				handleDeletionEvents(parser, msg, db, storage, activityLogger, trashRetentionDays)

			case eventparser.BucketEventTypeIgnore:
				zap.L().Debug("ignoring event", zap.String("raw_payload", string(msg.Payload)))

			default:
				zap.L().Warn("Unknown bucket event type", zap.String("payload", string(msg.Payload)))
			}

			msg.Ack()
		}
	}
}
