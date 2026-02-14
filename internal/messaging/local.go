package messaging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"go.uber.org/zap"
)

type LocalPublisher struct {
	topicName string
	channel   *gochannel.GoChannel
}

type LocalSubscriber struct {
	topicName string
	channel   *gochannel.GoChannel
}

func NewLocalChannel() *gochannel.GoChannel {
	return gochannel.NewGoChannel(gochannel.Config{
		Persistent: true,
	}, watermill.NopLogger{})
}

func NewLocalPublisher(channel *gochannel.GoChannel, topicName string) IPublisher {
	return &LocalPublisher{topicName: topicName, channel: channel}
}

func NewLocalSubscriber(channel *gochannel.GoChannel, topicName string) ISubscriber {
	return &LocalSubscriber{topicName: topicName, channel: channel}
}

func (p *LocalPublisher) Publish(messages ...*message.Message) error {
	return p.channel.Publish(p.topicName, messages...)
}

func (p *LocalPublisher) Close() error {
	return p.channel.Close()
}

func (s *LocalSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.channel.Subscribe(context.Background(), s.topicName)
	if err != nil {
		zap.L().Error("Failed to subscribe to local topic", zap.String("topic", s.topicName), zap.Error(err))
		return nil
	}
	return sub
}

func (s *LocalSubscriber) Close() error {
	return s.channel.Close()
}
