package messaging

import (
	"github.com/ThreeDotsLabs/watermill/message"
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
	ParseBucketUploadEvents(*message.Message) []BucketUploadEvent
	ParseBucketDeletionEvents(*message.Message) []BucketDeletionEvent
}
