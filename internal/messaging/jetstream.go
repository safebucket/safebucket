package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
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

// GetBucketEventType determines the type of MinIO S3 event.
func (s *JetStreamSubscriber) GetBucketEventType(message *message.Message) string {
	var event RustFSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(event.Records) == 0 {
		return BucketEventTypeUnknown
	}

	// Check the first record's event name to determine type
	eventName := event.Records[0].EventName
	objectKey := event.Records[0].Data.S3.Object.Key

	// Decode URL-encoded object key
	decodedKey, err := url.QueryUnescape(objectKey)
	if err != nil {
		zap.L().Debug("Failed to URL decode object key, using raw key",
			zap.String("raw_key", objectKey),
			zap.Error(err))
		decodedKey = objectKey
	}

	if eventName == "s3:ObjectCreated:Post" || eventName == "s3:ObjectCreated:Put" {
		// Exclude trash marker creation events - they lack metadata and should not be processed as uploads
		if strings.HasPrefix(decodedKey, "trash/") {
			zap.L().Debug("Ignoring trash marker creation event",
				zap.String("event_name", eventName),
				zap.String("object_key", decodedKey))
			return BucketEventTypeIgnore
		}
		return BucketEventTypeUpload
	}

	if strings.HasPrefix(eventName, "s3:ObjectRemoved:") ||
		strings.HasPrefix(eventName, "s3:LifecycleExpiration:") {
		return BucketEventTypeDeletion
	}

	// Log unhandled event types for debugging
	zap.L().Debug("Unrecognized S3 event type",
		zap.String("event_name", eventName),
		zap.String("raw_payload", string(message.Payload)))

	return BucketEventTypeIgnore
}

func (s *JetStreamSubscriber) ParseBucketUploadEvents(
	message *message.Message,
) []BucketUploadEvent {
	var event RustFSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		message.Ack()
		return nil
	}

	var uploadEvents []BucketUploadEvent
	for _, record := range event.Records {
		metadata := record.Data.S3.Object.UserMetadata

		// RustFS uses lowercase keys without X-Amz-Meta- prefix
		bucketID := metadata["bucket-id"]
		fileID := metadata["file-id"]
		userID := metadata["user-id"]

		uploadEvents = append(uploadEvents, BucketUploadEvent{
			BucketID: bucketID,
			FileID:   fileID,
			UserID:   userID,
		})
	}

	if len(uploadEvents) > 0 {
		message.Ack()
	}

	return uploadEvents
}

func (s *JetStreamSubscriber) ParseBucketDeletionEvents(
	message *message.Message,
	expectedBucketName string,
) []BucketDeletionEvent {
	var event RustFSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		message.Ack()
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		if record.Data.S3.Bucket.Name != expectedBucketName {
			zap.L().Debug("ignoring event from different bucket",
				zap.String("event_bucket", record.Data.S3.Bucket.Name),
				zap.String("expected_bucket", expectedBucketName))
			continue
		}

		objectKey, err := url.QueryUnescape(record.Data.S3.Object.Key)
		if err != nil {
			zap.L().Warn("failed to URL decode object key",
				zap.String("raw_key", record.Data.S3.Object.Key),
				zap.Error(err))
			objectKey = record.Data.S3.Object.Key // Fall back to raw key
		}

		zap.L().Debug("received deletion/expiration event",
			zap.String("event_name", record.EventName),
			zap.String("object_key", objectKey),
			zap.String("raw_payload", string(message.Payload)),
			zap.Any("user_metadata", record.Data.S3.Object.UserMetadata),
			zap.String("bucket_name", record.Data.S3.Bucket.Name),
			zap.Int64("size", record.Data.S3.Object.Size))

		var bucketID string

		// Handle different path patterns:
		// - buckets/{bucket-id}/{resource-id}
		// - trash/{bucket-id}/files/{file-id}
		// - trash/{bucket-id}/folders/{folder-id}
		if strings.HasPrefix(objectKey, "buckets/") {
			parts := strings.Split(objectKey, "/")
			if len(parts) >= 2 {
				bucketID = parts[1]
			}
		} else if strings.HasPrefix(objectKey, "trash/") {
			parts := strings.Split(objectKey, "/")
			if len(parts) >= 2 {
				bucketID = parts[1]
			}
		}

		if bucketID == "" {
			zap.L().Warn("unable to extract bucket ID from object key",
				zap.String("object_key", objectKey),
				zap.String("event_name", record.EventName))
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

	if len(deletionEvents) > 0 {
		message.Ack()
	}

	return deletionEvents
}
