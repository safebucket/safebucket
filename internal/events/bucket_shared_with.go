package events

import (
	"api/internal/messaging"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const BucketSharedWithName = "BucketSharedWith"
const BucketSharedWithPayloadName = "BucketSharedWithPayload"

type BucketSharedWithPayload struct {
	Type   string
	Emails []string
}

type BucketSharedWith struct {
	Publisher messaging.IPublisher
	Payload   BucketSharedWithPayload
}

func NewBucketSharedWith(publisher messaging.IPublisher, emails []string) BucketSharedWith {
	return BucketSharedWith{
		Publisher: publisher,
		Payload: BucketSharedWithPayload{
			Type:   BucketSharedWithName,
			Emails: emails,
		},
	}
}

func (e *BucketSharedWith) Trigger() {
	payload, _ := json.Marshal(e.Payload)
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err := e.Publisher.Publish("safebucket-notifications", msg)

	if err != nil {
		zap.L().Error("failed to trigger event", zap.Error(err))
	}
}

func (e *BucketSharedWith) callback() {
	zap.L().Info("message received", zap.Any("payload", e.Payload))
}
