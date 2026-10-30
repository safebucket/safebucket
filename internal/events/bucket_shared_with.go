package events

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
)

const (
	BucketSharedWithName        = "BucketSharedWith"
	BucketSharedWithPayloadName = "BucketSharedWithPayload"
)

type BucketSharedWithPayload struct {
	Type   string
	Bucket models.Bucket
	From   string
	To     string
	WebURL string
}

type BucketSharedWith struct {
	Payload BucketSharedWithPayload
}

func NewBucketSharedWith(
	bucket models.Bucket,
	from string,
	to string,
) BucketSharedWith {
	return BucketSharedWith{
		Payload: BucketSharedWithPayload{
			Type:   BucketSharedWithName,
			Bucket: bucket,
			From:   from,
			To:     to,
		},
	}
}

func (e *BucketSharedWith) callback(params *EventParams) error {
	e.Payload.WebURL = params.WebURL
	subject := fmt.Sprintf("%s has shared a bucket with you", e.Payload.From)
	err := params.Notifier.NotifyFromTemplate(
		e.Payload.To,
		subject,
		"bucket_shared_with",
		e.Payload,
	)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
