package core

import (
	"api/internal/messaging"
	"api/internal/models"
)

func NewPublisher(config models.EventsConfiguration) messaging.IPublisher {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamPublisher(config)
	default:
		return nil
	}
}

func NewSubscriber(config models.EventsConfiguration, topics []string) messaging.ISubscriber {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamSubscriber(config, topics)
	default:
		return nil
	}
}
