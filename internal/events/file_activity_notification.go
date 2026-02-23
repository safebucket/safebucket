package events

import (
	"encoding/json"
	"fmt"

	"api/internal/messaging"
	"api/internal/rbac"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FileActivityType string

const (
	FileActivityUpload   FileActivityType = "upload"
	FileActivityDownload FileActivityType = "download"
)

const (
	FileActivityNotificationName        = "FileActivityNotification"
	FileActivityNotificationPayloadName = "FileActivityNotificationPayload"
)

type FileActivityNotificationPayload struct {
	Type             string           `json:"type"`
	NotificationType FileActivityType `json:"notification_type"`
	BucketID         uuid.UUID        `json:"bucket_id"`
	BucketName       string           `json:"bucket_name"`
	FileName         string           `json:"file_name"`
	ActorID          uuid.UUID        `json:"actor_id"`
	ActorEmail       string           `json:"actor_email"`
}

type FileActivityNotification struct {
	Publisher messaging.IPublisher
	Payload   FileActivityNotificationPayload
}

func NewFileActivityNotification(
	publisher messaging.IPublisher,
	notificationType FileActivityType,
	bucketID uuid.UUID,
	bucketName string,
	fileName string,
	actorID uuid.UUID,
	actorEmail string,
) FileActivityNotification {
	return FileActivityNotification{
		Publisher: publisher,
		Payload: FileActivityNotificationPayload{
			Type:             FileActivityNotificationName,
			NotificationType: notificationType,
			BucketID:         bucketID,
			BucketName:       bucketName,
			FileName:         fileName,
			ActorID:          actorID,
			ActorEmail:       actorEmail,
		},
	}
}

func (e *FileActivityNotification) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger event", zap.Error(err))
	}
}

func (e *FileActivityNotification) callback(params *EventParams) error {
	memberships, err := rbac.GetBucketMembers(params.DB, e.Payload.BucketID)
	if err != nil {
		zap.L().Error("failed to get bucket members", zap.Error(err))
		return err
	}

	for _, m := range memberships {
		if m.UserID == e.Payload.ActorID {
			continue
		}
		if e.Payload.NotificationType == FileActivityUpload && !m.UploadNotifications {
			continue
		}
		if e.Payload.NotificationType == FileActivityDownload && !m.DownloadNotifications {
			continue
		}

		groupKey := batchGroupKey(m.User.Email, e.Payload.BucketID, e.Payload.ActorEmail, e.Payload.NotificationType)
		meta := batchMeta{
			RecipientEmail:   m.User.Email,
			ActorEmail:       e.Payload.ActorEmail,
			BucketID:         e.Payload.BucketID,
			BucketName:       e.Payload.BucketName,
			NotificationType: string(e.Payload.NotificationType),
			WebURL:           params.WebURL,
		}

		count, batchErr := addToBuffer(params.Cache, groupKey, e.Payload.FileName, meta)
		if batchErr != nil {
			return fmt.Errorf("failed to add to notification buffer: %w", batchErr)
		}

		zap.L().Debug("notification batched",
			zap.String("groupKey", groupKey),
			zap.Int64("count", count))
	}

	return nil
}
