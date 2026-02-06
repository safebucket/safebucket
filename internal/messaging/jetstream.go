package messaging

import (
	"context"
	"fmt"
	"net"
	"time"

	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	natsJs "github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type JetStreamPublisher struct {
	TopicName string
	publisher *jetstream.Publisher
}

func NewJetStreamPublisher(config *models.JetStreamEventsConfig, topicName string) IPublisher {
	nc, err := nats.Connect(net.JoinHostPort(config.Host, config.Port))
	if err != nil {
		zap.L().Fatal("Failed to connect to NATS", zap.Error(err))
	}

	publisher, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn: nc,
	})
	if err != nil {
		zap.L().Fatal("Failed to create JetStream publisher", zap.Error(err))
	}

	return &JetStreamPublisher{TopicName: topicName, publisher: publisher}
}

func (p *JetStreamPublisher) Publish(messages ...*message.Message) error {
	return p.publisher.Publish(p.TopicName, messages...)
}

func (p *JetStreamPublisher) Close() error {
	return p.publisher.Close()
}

type JetStreamSubscriber struct {
	TopicName  string
	subscriber *jetstream.Subscriber
}

func NewJetStreamSubscriber(config *models.JetStreamEventsConfig, topicName string) ISubscriber {
	nc, err := nats.Connect(net.JoinHostPort(config.Host, config.Port))
	if err != nil {
		zap.L().Fatal("Failed to connect to NATS", zap.Error(err))
	}

	js, err := natsJs.New(nc)
	if err != nil {
		zap.L().Fatal("Failed to create JetStream context", zap.Error(err))
	}

	stream, err := js.CreateStream(context.Background(), natsJs.StreamConfig{
		Name:      topicName,
		Subjects:  []string{topicName},
		Retention: natsJs.WorkQueuePolicy,
	})
	if err != nil {
		zap.L().Fatal("Failed to create stream",
			zap.String("stream_name", topicName),
			zap.String("subject", topicName),
			zap.Error(err))
	}

	consumerName := fmt.Sprintf("watermill__%s", topicName)
	_, err = stream.CreateOrUpdateConsumer(context.Background(), natsJs.ConsumerConfig{
		Name:      consumerName,
		AckPolicy: natsJs.AckExplicitPolicy,
	})
	if err != nil {
		zap.L().Fatal("Failed to create consumer",
			zap.String("consumer_name", consumerName),
			zap.Error(err))
	}

	var namer jetstream.ConsumerConfigurator
	subscriber, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:                nc,
		AckWaitTimeout:      5 * time.Second,
		ResourceInitializer: jetstream.ExistingConsumer(namer, ""),
		Logger:              watermill.NopLogger{},
	})
	if err != nil {
		zap.L().Fatal("Failed to create JetStream subscriber", zap.Error(err))
	}

	return &JetStreamSubscriber{TopicName: topicName, subscriber: subscriber}
}

func (s *JetStreamSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(context.Background(), s.TopicName)
	if err != nil {
		zap.L().
			Fatal("Failed to subscribe to topic", zap.String("topic", s.TopicName), zap.Error(err))
	}
	return sub
}

func (s *JetStreamSubscriber) Close() error {
	return s.subscriber.Close()
}
