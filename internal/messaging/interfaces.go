package messaging

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
)

// IPublisher defines a common interface for all publishers.
type IPublisher interface {
	Publish(topic string, messages ...*message.Message) error
	Close() error
}

// ISubscriber defines a common interface for all subscribers.
type ISubscriber interface {
	Subscribe(ctx context.Context, topic string) <-chan *message.Message
	Close() error
}
