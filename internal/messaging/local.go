package messaging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"go.uber.org/zap"
)

type MemoryPublisher struct {
	topicName string
	channel   *gochannel.GoChannel
}

type MemorySubscriber struct {
	topicName string
	channel   *gochannel.GoChannel
}

func NewMemoryChannel() *gochannel.GoChannel {
	return gochannel.NewGoChannel(gochannel.Config{
		Persistent: true,
	}, watermill.NopLogger{})
}

func NewMemoryPublisher(channel *gochannel.GoChannel, topicName string) IPublisher {
	return &MemoryPublisher{topicName: topicName, channel: channel}
}

func NewMemorySubscriber(channel *gochannel.GoChannel, topicName string) ISubscriber {
	return &MemorySubscriber{topicName: topicName, channel: channel}
}

func (p *MemoryPublisher) Publish(messages ...*message.Message) error {
	return p.channel.Publish(p.topicName, messages...)
}

func (p *MemoryPublisher) Close() error {
	return p.channel.Close()
}

func (s *MemorySubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.channel.Subscribe(context.Background(), s.topicName)
	if err != nil {
		zap.L().Error("Failed to subscribe to memory topic", zap.String("topic", s.topicName), zap.Error(err))
		return nil
	}
	return sub
}

func (s *MemorySubscriber) Close() error {
	return s.channel.Close()
}
