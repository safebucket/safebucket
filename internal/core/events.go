package core

import (
	"api/internal/messaging"
	"api/internal/models"
)

func NewPublisher(config models.EventsConfiguration) messaging.IPublisher {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamPublisher(config.Jetstream, config.TopicName)
	case "gcp":
		return messaging.NewGCPPublisher(config.GCP, config.TopicName)
	default:
		return nil
	}
}

func NewSubscriber(config models.EventsConfiguration) messaging.ISubscriber {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamSubscriber(config.Jetstream, config.TopicName)
	case "gcp":
		return messaging.NewGCPSubscriber(config.GCP)
	default:
		return nil
	}
}

func NewBucketEventsSubscriber(config models.StorageConfiguration) messaging.ISubscriber {
	switch config.Type {
	case "minio":
		switch config.Minio.Type {
		case "jetstream":
			return messaging.NewJetStreamSubscriber(config.Minio.Jetstream, config.TopicName)
		default:
			return nil
		}
	case "gcp":
		return messaging.NewGCPSubscriber(config.GCP)
	default:
		return nil
	}
}
