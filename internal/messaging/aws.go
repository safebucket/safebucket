package messaging

import (
	"context"

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
}

func NewAWSSubscriber(sqsName string) ISubscriber {
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

	return &AWSSubscriber{TopicName: sqsName, subscriber: subscriber}
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
