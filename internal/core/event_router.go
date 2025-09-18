package core

import (
	"api/internal/configuration"
	"api/internal/events"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type EventRouter struct {
	eventsManager *EventsManager
}

func NewEventRouter(eventsManager *EventsManager) *EventRouter {
	return &EventRouter{
		eventsManager: eventsManager,
	}
}

func (er *EventRouter) Publish(messages ...*message.Message) error {
	for _, msg := range messages {
		eventType := msg.Metadata.Get("type")
		topicKey := er.getTopicKeyForEvent(eventType)

		if topicKey == "" {
			zap.L().Fatal("Assign a topic to this event in getTopicKeyForEvent", zap.String("eventType", eventType))
		}

		publisher := er.eventsManager.GetPublisher(topicKey)
		if publisher == nil {
			continue
		}

		if err := publisher.Publish(msg); err != nil {
			return err
		}
	}

	return nil
}

func (er *EventRouter) Close() error {
	// EventsManager handles closing, so we don't need to do anything here
	return nil
}

// getTopicKeyForEvent maps event types to topic keys
func (er *EventRouter) getTopicKeyForEvent(eventType string) string {
	switch eventType {
	case events.UserInvitationName, events.BucketSharedWithName, events.ChallengeUserInviteName:
		return configuration.EventsNotifications
	case events.ObjectDeletionName:
		return configuration.EventsObjectDeletion
	default:
		return ""
	}
}
