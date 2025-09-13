package events

import (
	"api/internal/messaging"
	"api/internal/models"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const BucketSharedWithName = "BucketSharedWith"
const BucketSharedWithPayloadName = "BucketSharedWithPayload"

type BucketSharedWithPayload struct {
	Type   string
	Bucket models.Bucket
	From   string
	To     string
	WebUrl string
}

type BucketSharedWith struct {
	Publisher messaging.IPublisher
	Payload   BucketSharedWithPayload
}

func NewBucketSharedWith(
	publisher messaging.IPublisher,
	bucket models.Bucket,
	from string,
	to string,
) BucketSharedWith {
	return BucketSharedWith{
		Publisher: publisher,
		Payload: BucketSharedWithPayload{
			Type:   BucketSharedWithName,
			Bucket: bucket,
			From:   from,
			To:     to,
		},
	}
}

func (e *BucketSharedWith) Trigger() {
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

func (e *BucketSharedWith) callback(params *EventParams) error {
	e.Payload.WebUrl = params.WebUrl
	subject := fmt.Sprintf("%s has shared a bucket with you", e.Payload.From)
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "bucket_shared_with", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
