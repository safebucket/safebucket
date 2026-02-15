package core

import (
	"api/internal/configuration"
	"api/internal/eventparser"
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/storage"

	"go.uber.org/zap"
)

type EventsManager struct {
	publishers  map[string]messaging.IPublisher
	subscribers map[string]messaging.ISubscriber
	config      models.EventsConfiguration
	storage     storage.IStorage
	parser      eventparser.IBucketEventParser
}

func NewEventsManager(config models.EventsConfiguration, storageType string, storage storage.IStorage) *EventsManager {
	manager := &EventsManager{
		publishers:  make(map[string]messaging.IPublisher),
		subscribers: make(map[string]messaging.ISubscriber),
		config:      config,
		storage:     storage,
		parser:      eventparser.NewBucketEventParser(storageType, storage),
	}

	manager.initializePublishers()
	manager.initializeSubscribers()

	return manager
}

func (em *EventsManager) initializePublishers() {
	for topicKey, topicConfig := range em.config.Queues {
		var publisher messaging.IPublisher

		switch em.config.Type {
		case configuration.ProviderJetstream:
			publisher = messaging.NewJetStreamPublisher(&models.JetStreamEventsConfig{
				Host: em.config.Jetstream.Host,
				Port: em.config.Jetstream.Port,
			}, topicConfig.Name)
		case configuration.ProviderGCP:
			publisher = messaging.NewGCPPublisher(&models.PubSubConfiguration{
				ProjectID:          em.config.PubSub.ProjectID,
				SubscriptionSuffix: em.config.PubSub.SubscriptionSuffix,
			}, topicConfig.Name)
		case configuration.ProviderAWS:
			publisher = messaging.NewAWSPublisher(topicConfig.Name)
		case configuration.ProviderMemory:
			// Memory provider requires the same GoChannel instance to be shared between publisher and subscriber,
			// so both are created here. initializeSubscribers() skips the memory provider accordingly.
			ch := messaging.NewMemoryChannel()
			publisher = messaging.NewMemoryPublisher(ch, topicConfig.Name)
			em.subscribers[topicKey] = messaging.NewMemorySubscriber(ch, topicConfig.Name)
		}

		em.publishers[topicKey] = publisher

		zap.L().Info("Initialized publisher",
			zap.String("topic_key", topicKey),
			zap.String("topic_name", topicConfig.Name),
			zap.String("provider", em.config.Type))
	}
}

func (em *EventsManager) initializeSubscribers() {
	for topicKey, topicConfig := range em.config.Queues {
		var subscriber messaging.ISubscriber

		switch em.config.Type {
		case configuration.ProviderJetstream:
			subscriber = messaging.NewJetStreamSubscriber(&models.JetStreamEventsConfig{
				Host: em.config.Jetstream.Host,
				Port: em.config.Jetstream.Port,
			}, topicConfig.Name)
		case configuration.ProviderGCP:
			subscriber = messaging.NewGCPSubscriber(&models.PubSubConfiguration{
				ProjectID:          em.config.PubSub.ProjectID,
				SubscriptionSuffix: em.config.PubSub.SubscriptionSuffix,
			}, topicConfig.Name)
		case configuration.ProviderAWS:
			subscriber = messaging.NewAWSSubscriber(topicConfig.Name)
		case configuration.ProviderMemory:
			// Memory subscribers are already created in initializePublishers() (shared GoChannel).
			continue
		}

		if subscriber != nil {
			em.subscribers[topicKey] = subscriber
			zap.L().Info("Initialized subscriber",
				zap.String("topic_key", topicKey),
				zap.String("topic_name", topicConfig.Name),
				zap.String("provider", em.config.Type))
		}
	}
}

func (em *EventsManager) GetPublisher(topicKey string) messaging.IPublisher {
	publisher, exists := em.publishers[topicKey]
	if !exists {
		zap.L().Warn("Publisher not found", zap.String("topic_key", topicKey))
		return nil
	}
	return publisher
}

func (em *EventsManager) GetSubscriber(topicKey string) messaging.ISubscriber {
	subscriber, exists := em.subscribers[topicKey]
	if !exists {
		zap.L().Warn("Subscriber not found", zap.String("topic_key", topicKey))
		return nil
	}
	return subscriber
}

func (em *EventsManager) Close() {
	for topicKey, publisher := range em.publishers {
		if err := publisher.Close(); err != nil {
			zap.L().Error("Failed to close publisher",
				zap.String("topic_key", topicKey),
				zap.Error(err))
		}
	}

	for topicKey, subscriber := range em.subscribers {
		if err := subscriber.Close(); err != nil {
			zap.L().Error("Failed to close subscriber",
				zap.String("topic_key", topicKey),
				zap.Error(err))
		}
	}
}
