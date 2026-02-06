package messaging

import (
	"context"

	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type GCPPublisher struct {
	TopicName string
	publisher *googlecloud.Publisher
}

func NewGCPPublisher(config *models.PubSubConfiguration, topicName string) IPublisher {
	publisher, err := googlecloud.NewPublisher(googlecloud.PublisherConfig{
		ProjectID: config.ProjectID,
	}, nil)
	if err != nil {
		zap.L().Fatal("Failed to create PUB/SUB publisher", zap.Error(err))
	}

	return &GCPPublisher{TopicName: topicName, publisher: publisher}
}

func (p *GCPPublisher) Publish(messages ...*message.Message) error {
	return p.publisher.Publish(p.TopicName, messages...)
}

func (p *GCPPublisher) Close() error {
	return p.publisher.Close()
}

type GCPSubscriber struct {
	TopicName  string
	subscriber *googlecloud.Subscriber
}

func NewGCPSubscriber(config *models.PubSubConfiguration, topicName string) ISubscriber {
	subscriber, err := googlecloud.NewSubscriber(
		googlecloud.SubscriberConfig{
			ProjectID: config.ProjectID,
			GenerateSubscriptionName: func(topic string) string {
				return topic + config.SubscriptionSuffix
			},
			DoNotCreateSubscriptionIfMissing: true,
		},
		nil,
	)
	if err != nil {
		zap.L().Fatal("Failed to create PUB/SUB subscriber", zap.Error(err))
	}

	return &GCPSubscriber{TopicName: topicName, subscriber: subscriber}
}

func (s *GCPSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(context.Background(), s.TopicName)
	if err != nil {
		zap.L().
			Fatal("Failed to subscribe to topic", zap.String("topic", s.TopicName), zap.Error(err))
	}
	return sub
}

func (s *GCPSubscriber) Close() error {
	return s.subscriber.Close()
}
