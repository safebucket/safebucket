package messaging

import (
	"api/internal/models"
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	natsJs "github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"time"
)

type JetStreamPublisher struct {
	TopicName string
	publisher *jetstream.Publisher
}

func NewJetStreamPublisher(config *models.JetStreamEventsConfig) IPublisher {
	nc, _ := nats.Connect(fmt.Sprintf("nats://%s:%s", config.Host, config.Port))

	publisher, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn: nc,
	})
	if err != nil {
		zap.L().Fatal("Failed to create JetStream publisher", zap.Error(err))
	}

	return &JetStreamPublisher{TopicName: config.TopicName, publisher: publisher}
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

func NewJetStreamSubscriber(config *models.JetStreamEventsConfig) ISubscriber {
	nc, _ := nats.Connect(fmt.Sprintf("nats://%s:%s", config.Host, config.Port))
	js, _ := natsJs.New(nc)

	stream, _ := js.CreateStream(context.Background(), natsJs.StreamConfig{
		Name:      config.TopicName,
		Subjects:  []string{config.TopicName},
		Retention: natsJs.WorkQueuePolicy,
	})

	consumerName := fmt.Sprintf("watermill__%s", config.TopicName)
	_, _ = stream.CreateOrUpdateConsumer(context.Background(), natsJs.ConsumerConfig{
		Name:      consumerName,
		AckPolicy: natsJs.AckExplicitPolicy,
	})

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

	return &JetStreamSubscriber{TopicName: config.TopicName, subscriber: subscriber}
}

func (s *JetStreamSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(context.Background(), s.TopicName)
	if err != nil {
		zap.L().Fatal("Failed to subscribe to topic", zap.String("topic", s.TopicName), zap.Error(err))
	}
	return sub
}

func (s *JetStreamSubscriber) Close() error {
	return s.subscriber.Close()
}
