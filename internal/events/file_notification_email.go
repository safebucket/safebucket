package events

import (
	"encoding/json"
	"fmt"

	"api/internal/messaging"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	FileNotificationEmailName        = "FileNotificationEmail"
	FileNotificationEmailPayloadName = "FileNotificationEmailPayload"
)

type FileNotificationEmailPayload struct {
	Type             string
	NotificationType FileActivityType
	RecipientEmail   string
	ActorEmail       string
	FileName         string
	BucketID         uuid.UUID
	BucketName       string
	WebURL           string
	ActionText       string
}

type FileNotificationEmail struct {
	Publisher messaging.IPublisher
	Payload   FileNotificationEmailPayload
}

func NewFileNotificationEmail(
	publisher messaging.IPublisher,
	notificationType FileActivityType,
	recipientEmail string,
	actorEmail string,
	fileName string,
	bucketID uuid.UUID,
	bucketName string,
) FileNotificationEmail {
	return FileNotificationEmail{
		Publisher: publisher,
		Payload: FileNotificationEmailPayload{
			Type:             FileNotificationEmailName,
			NotificationType: notificationType,
			RecipientEmail:   recipientEmail,
			ActorEmail:       actorEmail,
			FileName:         fileName,
			BucketID:         bucketID,
			BucketName:       bucketName,
		},
	}
}

func (e *FileNotificationEmail) Trigger() {
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

func (e *FileNotificationEmail) callback(params *EventParams) error {
	e.Payload.WebURL = params.WebURL

	var action string
	if e.Payload.NotificationType == FileActivityUpload {
		action = "uploaded a file to"
		e.Payload.ActionText = fmt.Sprintf(
			"%s uploaded \"%s\" to bucket \"%s\".",
			e.Payload.ActorEmail,
			e.Payload.FileName,
			e.Payload.BucketName,
		)
	} else {
		action = "downloaded a file from"
		e.Payload.ActionText = fmt.Sprintf(
			"%s downloaded \"%s\" from bucket \"%s\".",
			e.Payload.ActorEmail,
			e.Payload.FileName,
			e.Payload.BucketName,
		)
	}

	subject := fmt.Sprintf("%s %s %s", e.Payload.ActorEmail, action, e.Payload.BucketName)
	err := params.Notifier.NotifyFromTemplate(
		e.Payload.RecipientEmail,
		subject,
		"file_activity",
		e.Payload,
	)
	if err != nil {
		zap.L().Error("failed to send file notification email",
			zap.String("to", e.Payload.RecipientEmail),
			zap.Error(err))
		return err
	}

	return nil
}
