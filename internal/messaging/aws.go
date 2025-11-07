package messaging

import (
	"context"
	"encoding/json"
	"strings"

	"api/internal/storage"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"go.uber.org/zap"
)

type AWSPublisher struct {
	TopicName string
	publisher *sqs.Publisher
}

func NewAWSPublisher(queueName string) IPublisher {
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		zap.L().Fatal("Unable to load SDK config.", zap.Error(err))
	}

	if err != nil {
		zap.L().Error("Unable to retrieve AWS credentials.", zap.Error(err))
	}

	publisher, err := sqs.NewPublisher(sqs.PublisherConfig{
		AWSConfig:                   awsCfg,
		DoNotCreateQueueIfNotExists: true,
		Marshaler:                   sqs.DefaultMarshalerUnmarshaler{},
	}, watermill.NopLogger{})
	if err != nil {
		zap.L().Fatal("Unable to create publisher", zap.Error(err))
	}

	return &AWSPublisher{TopicName: queueName, publisher: publisher}
}

func (p *AWSPublisher) Publish(messages ...*message.Message) error {
	return p.publisher.Publish(p.TopicName, messages...)
}

func (p *AWSPublisher) Close() error {
	return p.publisher.Close()
}

type AWSSubscriber struct {
	TopicName  string
	subscriber *sqs.Subscriber
	storage    storage.IStorage
}

func NewAWSSubscriber(sqsName string, storage storage.IStorage) ISubscriber {
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		zap.L().Fatal("Unable to load SDK config.", zap.Error(err))
	}

	subscriber, err := sqs.NewSubscriber(sqs.SubscriberConfig{
		AWSConfig:                   awsCfg,
		DoNotCreateQueueIfNotExists: true,
	}, nil)
	if err != nil {
		zap.L().Fatal("Failed to create CloudStorage subscriber", zap.Error(err))
	}

	return &AWSSubscriber{TopicName: sqsName, subscriber: subscriber, storage: storage}
}

func (s *AWSSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(context.Background(), s.TopicName)
	if err != nil {
		zap.L().
			Fatal("Failed to subscribe to topic", zap.String("topic", s.TopicName), zap.Error(err))
	}
	return sub
}

func (s *AWSSubscriber) Close() error {
	return s.subscriber.Close()
}

// GetBucketEventType determines the type of AWS S3 event
func (s *AWSSubscriber) GetBucketEventType(message *message.Message) string {
	var event AWSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("Failed to unmarshal event to determine type", zap.Error(err))
		return BucketEventTypeUnknown
	}

	if len(event.Records) == 0 {
		return BucketEventTypeUnknown
	}

	eventName := event.Records[0].EventName

	if eventName == "ObjectCreated:Post" || eventName == "ObjectCreated:Put" {
		return BucketEventTypeUpload
	}

	if strings.HasPrefix(eventName, "ObjectRemoved:") ||
		strings.HasPrefix(eventName, "LifecycleExpiration:") {
		return BucketEventTypeDeletion
	}

	return BucketEventTypeUnknown
}

func (s *AWSSubscriber) ParseBucketUploadEvents(message *message.Message) []BucketUploadEvent {
	var event AWSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		message.Ack()
	}

	var uploadEvents []BucketUploadEvent
	for _, record := range event.Records {
		if record.EventName == "ObjectCreated:Post" || record.EventName == "ObjectCreated:Put" {
			metadata, err := s.storage.StatObject(record.S3.Object.Key)
			if err != nil {
				zap.L().Error("failed to stat object", zap.Error(err))
			}

			bucketID := metadata["bucket_id"]
			fileID := metadata["file_id"]
			userID := metadata["user_id"]

			uploadEvents = append(uploadEvents, BucketUploadEvent{
				BucketID: bucketID,
				FileID:   fileID,
				UserID:   userID,
			})
			message.Ack()
		}
	}

	return uploadEvents
}

func (s *AWSSubscriber) ParseBucketDeletionEvents(message *message.Message, expectedBucketName string) []BucketDeletionEvent {
	var event AWSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		message.Ack()
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		eventName := record.EventName
		isRemoveEvent := strings.HasPrefix(eventName, "ObjectRemoved:")
		isLifecycleEvent := strings.HasPrefix(eventName, "LifecycleExpiration:")

		if isRemoveEvent || isLifecycleEvent {
			if record.S3.Bucket.Name != expectedBucketName {
				zap.L().Debug("ignoring event from different bucket",
					zap.String("event_bucket", record.S3.Bucket.Name),
					zap.String("expected_bucket", expectedBucketName))
				continue
			}

			objectKey := record.S3.Object.Key
			var bucketID string

			// Handle both "buckets/" and "trash/" prefixes
			if strings.HasPrefix(objectKey, "buckets/") || strings.HasPrefix(objectKey, "trash/") {
				parts := strings.Split(objectKey, "/")
				if len(parts) >= 2 {
					bucketID = parts[1]
				}
			}

			if bucketID == "" {
				zap.L().Warn("unable to extract bucket ID from object key",
					zap.String("object_key", objectKey),
					zap.String("event_name", eventName))
				continue
			}

			deletionEvents = append(deletionEvents, BucketDeletionEvent{
				BucketID:  bucketID,
				ObjectKey: objectKey,
				EventName: eventName,
			})

			zap.L().Debug("parsed deletion event",
				zap.String("event_name", eventName),
				zap.String("bucket_id", bucketID),
				zap.String("object_key", objectKey))
		}
	}

	if len(deletionEvents) > 0 {
		message.Ack()
	}

	return deletionEvents
}
