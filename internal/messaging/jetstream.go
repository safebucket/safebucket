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
	publisher *jetstream.Publisher
}

func NewJetStreamPublisher(config models.EventsConfiguration) IPublisher {
	nc, _ := nats.Connect(fmt.Sprintf("nats://%s:%s", config.Host, config.Port))

	publisher, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn: nc,
	})
	if err != nil {
		zap.L().Fatal("Failed to create JetStream publisher", zap.Error(err))
	}

	return &JetStreamPublisher{publisher: publisher}
}

func (p *JetStreamPublisher) Publish(topic string, messages ...*message.Message) error {
	return p.publisher.Publish(topic, messages...)
}

func (p *JetStreamPublisher) Close() error {
	return p.publisher.Close()
}

type JetStreamSubscriber struct {
	subscriber *jetstream.Subscriber
}

func NewJetStreamSubscriber(config models.EventsConfiguration, topics []string) ISubscriber {
	nc, _ := nats.Connect(fmt.Sprintf("nats://%s:%s", config.Host, config.Port))
	js, _ := natsJs.New(nc)

	for _, topic := range topics {
		stream, _ := js.CreateStream(context.Background(), natsJs.StreamConfig{
			Name:      topic,
			Subjects:  []string{topic},
			Retention: natsJs.WorkQueuePolicy,
		})

		consumerName := fmt.Sprintf("watermill__%s", topic)
		_, _ = stream.CreateOrUpdateConsumer(context.Background(), natsJs.ConsumerConfig{
			Name:      consumerName,
			AckPolicy: natsJs.AckExplicitPolicy,
		})
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

	return &JetStreamSubscriber{subscriber: subscriber}
}

func (s *JetStreamSubscriber) Subscribe(ctx context.Context, topic string) <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(ctx, topic)
	if err != nil {
		zap.L().Fatal("Failed to subscribe to topic", zap.String("topic", topic), zap.Error(err))
	}
	return sub
}

func (s *JetStreamSubscriber) Close() error {
	return s.subscriber.Close()
}
