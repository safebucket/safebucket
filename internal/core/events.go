package core

import (
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/storage"
)

func NewPublisher(config models.EventsConfiguration) messaging.IPublisher {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamPublisher(config.Jetstream)
	case "pubsub":
		return messaging.NewGCPPublisher(config.PubSub)
	case "sqs":
		return messaging.NewAWSPublisher(config.SQS)
	default:
		return nil
	}
}

func NewSubscriber(config models.EventsConfiguration) messaging.ISubscriber {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamSubscriber(config.Jetstream)
	case "pubsub":
		return messaging.NewGCPSubscriber(config.PubSub)
	case "sqs":
		return messaging.NewAWSSubscriber(config.SQS.Name, nil)
	default:
		return nil
	}
}

func NewBucketEventsSubscriber(config models.StorageConfiguration, storage storage.IStorage) messaging.ISubscriber {
	switch config.Type {
	case "minio":
		switch config.Minio.Type {
		case "jetstream":
			return messaging.NewJetStreamSubscriber(config.Minio.Jetstream)
		default:
			return nil
		}
	case "cloud_storage":
		return messaging.NewGCPSubscriber(&models.PubSubConfiguration{
			ProjectID:        config.CloudStorage.ProjectID,
			SubscriptionName: config.CloudStorage.SubscriptionName,
			TopicName:        config.CloudStorage.TopicName})
	case "s3":
		return messaging.NewAWSSubscriber(config.S3.SQSName, storage)
	default:
		return nil
	}
}
