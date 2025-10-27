package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api/internal/models"

	"net"

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

func (s *JetStreamSubscriber) ParseBucketUploadEvents(
	message *message.Message,
) []BucketUploadEvent {
	var event MinioEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		message.Ack()
	}

	var uploadEvents []BucketUploadEvent
	for _, event := range event.Records {
		if event.EventName == "s3:ObjectCreated:Post" {
			bucketID := event.S3.Object.UserMetadata["X-Amz-Meta-Bucket-Id"]
			fileID := event.S3.Object.UserMetadata["X-Amz-Meta-File-Id"]
			userID := event.S3.Object.UserMetadata["X-Amz-Meta-User-Id"]

			uploadEvents = append(uploadEvents, BucketUploadEvent{
				BucketID: bucketID,
				FileID:   fileID,
				UserID:   userID,
			})

			message.Ack()
		} else {
			zap.L().Warn("event is not supported", zap.Any("event_name", event.EventName))
			message.Ack()
			continue
		}
	}

	return uploadEvents
}

func (s *JetStreamSubscriber) ParseBucketDeletionEvents(
	message *message.Message,
) []BucketDeletionEvent {
	var event MinioEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		message.Ack()
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		if strings.HasPrefix(record.EventName, "s3:ObjectRemoved:") ||
			strings.HasPrefix(record.EventName, "s3:LifecycleExpiration:") {
			objectKey := record.S3.Object.Key
			var bucketID string
			if strings.HasPrefix(objectKey, "buckets/") {
				parts := strings.Split(objectKey, "/")
				if len(parts) >= 2 {
					bucketID = parts[1]
				}
			}

			if bucketID == "" {
				zap.L().Warn("unable to extract bucket ID from object key",
					zap.String("object_key", objectKey))
				continue
			}

			deletionEvents = append(deletionEvents, BucketDeletionEvent{
				BucketID:  bucketID,
				ObjectKey: objectKey,
				EventName: record.EventName,
			})

			zap.L().Debug("parsed deletion event",
				zap.String("event_name", record.EventName),
				zap.String("bucket_id", bucketID),
				zap.String("object_key", objectKey))
		}
	}

	if len(deletionEvents) > 0 {
		message.Ack()
	}

	return deletionEvents
}
