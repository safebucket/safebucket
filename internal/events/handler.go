package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/eventparser"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/notifier"
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
	Versions           fileversions.Manager
}

type Event interface {
	callback(params *EventParams) error
}

func getEventFromMessage(eventType string, msg *message.Message) (Event, error) {
	payloadTypeName := fmt.Sprintf("%sPayload", eventType)
	payloadType, exists := eventRegistry[payloadTypeName]

	if !exists {
		return nil, fmt.Errorf("payload type %s not found in event registry", payloadTypeName)
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

func HandleEvents(ctx context.Context, workerName string, params *EventParams, messages <-chan *message.Message) {
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
				zap.L().Error("event is misconfigured",
					zap.Error(err),
					zap.String("eventType", eventType),
					zap.String("worker", workerName),
				)
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

func handleUploadEvents(parser eventparser.IBucketEventParser, msg *message.Message,
	manager fileversions.Manager, activityLogger activity.IActivityLogger, publisher messaging.IPublisher,
) error {
	for _, event := range parser.ParseBucketUploadEvents(msg) {
		bucketID, err := uuid.Parse(event.BucketID)
		if err != nil {
			zap.L().Warn("Invalid upload notification identifier", zap.String("field", "BucketID"), zap.Error(err))
			continue
		}
		fileID, err := uuid.Parse(event.FileID)
		if err != nil {
			zap.L().Warn("Invalid upload notification identifier", zap.String("field", "FileID"), zap.Error(err))
			continue
		}
		versionID := fileID
		if event.VersionID != "" {
			versionID, err = uuid.Parse(event.VersionID)
			if err != nil {
				zap.L().Warn("Invalid upload notification version", zap.Error(err))
				continue
			}
		}
		completed, err := manager.Complete(zap.L(), bucketID, fileID, versionID, nil, true)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) && apiErr.Status < 500 {
				zap.L().
					Warn("Upload notification rejected", zap.Error(err), zap.String("version_id", versionID.String()))
				continue
			}
			return err
		}
		if completed.Changed {
			NotifyFileVersionUploaded(zap.L(), manager.DB, activityLogger, publisher, completed)
		} else if event.VersionID == "" {
			zap.L().Warn("Upload notification without version metadata made no change",
				zap.String("file_id", fileID.String()), zap.String("bucket_id", bucketID.String()))
		}
	}
	return nil
}

func handleDeletionEvents(
	parser eventparser.IBucketEventParser,
	msg *message.Message,
	db *gorm.DB,
	storage storage.IStorage,
	activityLogger activity.IActivityLogger,
	trashRetentionDays int,
	versions fileversions.Manager,
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
			Versions:           versions,
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
	versions fileversions.Manager,
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
				if err := handleUploadEvents(parser, msg, versions, activityLogger, publisher); err != nil {
					zap.L().Error("Failed to confirm version from storage event", zap.Error(err))
					msg.Nack()
					continue
				}

			case eventparser.BucketEventTypeDeletion:
				handleDeletionEvents(parser, msg, db, storage, activityLogger, trashRetentionDays, versions)

			case eventparser.BucketEventTypeIgnore:
				zap.L().Debug("ignoring event", zap.String("raw_payload", string(msg.Payload)))

			default:
				zap.L().Warn("Unknown bucket event type", zap.String("payload", string(msg.Payload)))
			}

			msg.Ack()
		}
	}
}
