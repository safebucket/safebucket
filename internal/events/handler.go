package events

import (
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
	"reflect"
)

type Event interface {
	callback()
}

func getEventFromMessage(eventType string, msg *message.Message) (Event, error) {
	payloadType, exists := eventRegistry[fmt.Sprintf("%sPayload", eventType)]

	if !exists {
		return nil, fmt.Errorf("payload type %s not found in event registry", eventType)
	}

	payload := reflect.New(payloadType).Interface()

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message payload: %w", err)
	}

	eventTyp, exists := eventRegistry[eventType]
	if !exists {
		return nil, fmt.Errorf("event type %s not found in event registry", eventType)
	}

	eventInstance := reflect.New(eventTyp).Interface()

	eventValue := reflect.ValueOf(eventInstance).Elem()
	payloadField := eventValue.FieldByName("Payload")
	if !payloadField.IsValid() || !payloadField.CanSet() {
		return nil, fmt.Errorf("event type %s does not have a settable 'Payload' field", eventType)
	}
	payloadField.Set(reflect.ValueOf(payload).Elem())

	event, ok := eventInstance.(Event)
	if !ok {
		return nil, fmt.Errorf("type %s does not implement Event interface", eventType)
	}

	return event, nil
}

func HandleNotifications(messages <-chan *message.Message) {
	for msg := range messages {
		zap.L().Debug("message received", zap.Any("raw_payload", string(msg.Payload)), zap.Any("metadata", msg.Metadata))

		eventType := msg.Metadata.Get("type")
		event, err := getEventFromMessage(eventType, msg)

		if err != nil {
			zap.L().Error("event is misconfigured", zap.Error(err))
			msg.Ack()
			continue
		}

		event.callback()
		msg.Ack()
	}
}
