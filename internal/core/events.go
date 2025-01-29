package core

import (
	"api/internal/messaging"
	"api/internal/models"
)

func NewPublisher(config models.EventsConfiguration, topic string) messaging.IPublisher {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamPublisher(config, topic)
	default:
		return nil
	}
}

func NewSubscriber(config models.EventsConfiguration) messaging.ISubscriber {
	switch config.Type {
	case "jetstream":
		return messaging.NewJetStreamSubscriber(config)
	default:
		return nil
	}
}
