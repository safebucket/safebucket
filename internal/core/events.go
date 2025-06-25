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
	case "gcp":
		return messaging.NewGCPPublisher(config.GCP)
	case "aws":
		return messaging.NewAWSPublisher(config.AWS)
	default:
		return nil
	}
}

func NewSubscriber(config models.EventsConfiguration) messaging.ISubscriber {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamSubscriber(config.Jetstream)
	case "gcp":
		return messaging.NewGCPSubscriber(config.GCP)
	case "aws":
		return messaging.NewAWSSubscriber(config.AWS.SQSName, nil)
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
	case "gcp":
		return messaging.NewGCPSubscriber(config.GCP)
	case "aws":
		return messaging.NewAWSSubscriber(config.AWS.SQSName, storage)
	default:
		return nil
	}
}
