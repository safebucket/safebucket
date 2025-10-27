package messaging

import (
	"context"
	"encoding/json"

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

func (s *AWSSubscriber) ParseBucketUploadEvents(message *message.Message) []BucketUploadEvent {
	var event AWSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("event is unprocessable", zap.Error(err))
		message.Ack()
	}

	var uploadEvents []BucketUploadEvent
	for _, event := range event.Records {
		if event.EventName == "ObjectCreated:Post" {
			metadata, err := s.storage.StatObject(event.S3.Object.Key)
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
		} else {
			zap.L().Warn("event is not supported", zap.Any("event_name", event.EventName))
			message.Ack()
			continue
		}
	}

	return uploadEvents
}

func (s *AWSSubscriber) ParseBucketDeletionEvents(message *message.Message) []BucketDeletionEvent {
	var event AWSEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		zap.L().Error("deletion event is unprocessable", zap.Error(err))
		message.Ack()
		return nil
	}

	var deletionEvents []BucketDeletionEvent
	for _, record := range event.Records {
		eventName := record.EventName
		isRemoveEvent := len(eventName) >= len("ObjectRemoved:") &&
			eventName[:len("ObjectRemoved:")] == "ObjectRemoved:"
		isLifecycleEvent := len(eventName) >= len("LifecycleExpiration:") &&
			eventName[:len("LifecycleExpiration:")] == "LifecycleExpiration:"

		if isRemoveEvent || isLifecycleEvent {
			objectKey := record.S3.Object.Key
			var bucketID string
			if len(objectKey) > 8 && objectKey[:8] == "buckets/" {
				parts := make([]string, 0)
				start := 0
				for i, c := range objectKey {
					if c == '/' {
						parts = append(parts, objectKey[start:i])
						start = i + 1
					}
				}
				if start < len(objectKey) {
					parts = append(parts, objectKey[start:])
				}

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
