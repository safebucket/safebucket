package messaging

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Bucket event type constants.
const (
	BucketEventTypeUpload   = "upload"
	BucketEventTypeDeletion = "deletion"
	BucketEventTypeUnknown  = "unknown"
)

// IPublisher defines a common interface for all publishers.
type IPublisher interface {
	Publish(messages ...*message.Message) error
	Close() error
}

// ISubscriber defines a common interface for all subscribers.
type ISubscriber interface {
	Subscribe() <-chan *message.Message
	Close() error
	GetBucketEventType(*message.Message) string
	ParseBucketUploadEvents(*message.Message) []BucketUploadEvent
	ParseBucketDeletionEvents(*message.Message, string) []BucketDeletionEvent
}
