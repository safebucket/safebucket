package messaging

import (
	"api/internal/models"
	"context"
	"github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type GCPPublisher struct {
	TopicName string
	publisher *googlecloud.Publisher
}

func NewGCPPublisher(config *models.GCPConfiguration, topic string) IPublisher {
	publisher, err := googlecloud.NewPublisher(googlecloud.PublisherConfig{
		ProjectID: config.ProjectID,
	}, nil)

	if err != nil {
		zap.L().Fatal("Failed to create GCP publisher", zap.Error(err))
	}

	return &GCPPublisher{TopicName: topic, publisher: publisher}
}

func (p *GCPPublisher) Publish(messages ...*message.Message) error {
	return p.publisher.Publish(p.TopicName, messages...)
}

func (p *GCPPublisher) Close() error {
	return p.publisher.Close()
}

type GCPSubscriber struct {
	subscriber *googlecloud.Subscriber
}

func NewGCPSubscriber(config *models.GCPConfiguration) ISubscriber {
	subscriber, err := googlecloud.NewSubscriber(
		googlecloud.SubscriberConfig{
			ProjectID: config.ProjectID,
			GenerateSubscriptionName: func(topic string) string {
				return config.SubscriptionName
			},
			DoNotCreateSubscriptionIfMissing: true,
		},
		nil,
	)

	if err != nil {
		zap.L().Fatal("Failed to create GCP subscriber", zap.Error(err))
	}

	return &GCPSubscriber{subscriber: subscriber}
}

func (s *GCPSubscriber) Subscribe(ctx context.Context, topic string) <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(ctx, topic)
	if err != nil {
		zap.L().Fatal("Failed to subscribe to topic", zap.String("topic", topic), zap.Error(err))
	}
	return sub
}

func (s *GCPSubscriber) Close() error {
	return s.subscriber.Close()
}
