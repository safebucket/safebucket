package messaging

import (
	"api/internal/models"
	"api/internal/storage"
	"context"
	"encoding/json"
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

func NewAWSPublisher(config *models.AWSConfiguration) IPublisher {
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background())

	if err != nil {
		zap.L().Fatal("Unable to load SDK config.", zap.Error(err))
	}

	creds, err := awsCfg.Credentials.Retrieve(context.Background())

	if err != nil {
		zap.L().Error("Unable to retrieve AWS credentials.", zap.Error(err))
	}

	publisher, err := sqs.NewPublisher(sqs.PublisherConfig{
		AWSConfig: awsCfg,
		QueueUrlResolver: sqs.GenerateQueueUrlResolver{
			AwsRegion:    awsCfg.Region,
			AwsAccountID: creds.AccountID,
		},
		Marshaler: sqs.DefaultMarshalerUnmarshaler{},
	}, watermill.NopLogger{})

	if err != nil {
		zap.L().Fatal("Unable to create publisher", zap.Error(err))
	}

	return &AWSPublisher{TopicName: config.SQSName, publisher: publisher}
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
		AWSConfig: awsCfg,
	}, nil)

	if err != nil {
		zap.L().Fatal("Failed to create GCP subscriber", zap.Error(err))
	}

	return &AWSSubscriber{TopicName: sqsName, subscriber: subscriber, storage: storage}
}

func (s *AWSSubscriber) Subscribe() <-chan *message.Message {
	sub, err := s.subscriber.Subscribe(context.Background(), s.TopicName)
	if err != nil {
		zap.L().Fatal("Failed to subscribe to topic", zap.String("topic", s.TopicName), zap.Error(err))
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

			bucketId := metadata["bucket_id"]
			fileId := metadata["file_id"]
			userId := metadata["user_id"]

			uploadEvents = append(uploadEvents, BucketUploadEvent{
				BucketId: bucketId,
				FileId:   fileId,
				UserId:   userId,
			})
		} else {
			zap.L().Warn("event is not supported", zap.Any("event_name", event.EventName))
			continue
		}
	}

	return uploadEvents
}
