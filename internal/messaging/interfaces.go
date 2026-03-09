package messaging

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type IPublisher interface {
	Publish(messages ...*message.Message) error
	Close() error
}

type ISubscriber interface {
	Subscribe() <-chan *message.Message
	Close() error
}
