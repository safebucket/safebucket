package core

import (
	"api/internal/models"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"time"
)

var marshaler = &nats.JSONMarshaler{}

var subscribeOptions = []nc.SubOpt{
	nc.DeliverNew(),
	nc.AckExplicit(),
}

var jsConfig = nats.JetStreamConfig{
	Disabled:         false,
	AutoProvision:    true,
	ConnectOptions:   []nc.JSOpt{},
	SubscribeOptions: subscribeOptions,
	PublishOptions:   []nc.PubOpt{},
	TrackMsgId:       false,
	AckAsync:         false,
	DurablePrefix:    "",
}

var options = []nc.Option{
	nc.RetryOnFailedConnect(true),
	nc.Timeout(30 * time.Second),
	nc.ReconnectWait(1 * time.Second),
}

var logger = watermill.NewStdLogger(false, false)

func InitPublisher() *nats.Publisher {
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         "nats://127.0.0.1:4222",
			NatsOptions: options,
			Marshaler:   marshaler,
			JetStream:   jsConfig,
		},
		logger,
	)

	if err != nil {
		zap.L().Fatal("Failed to create nats publisher", zap.Error(err))
	}

	return publisher
}

func InitSubscriber() *nats.Subscriber {
	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            "nats://127.0.0.1:4222",
			CloseTimeout:   30 * time.Second,
			AckWaitTimeout: 30 * time.Second,
			NatsOptions:    options,
			Unmarshaler:    marshaler,
			JetStream:      jsConfig,
		},
		logger,
	)

	if err != nil {
		zap.L().Fatal("Failed to create nats subscriber", zap.Error(err))
	}

	return subscriber
}

func PublishEvent(publisher *nats.Publisher, event models.Event) {
	payload, _ := json.Marshal(event)
	msg := message.NewMessage(watermill.NewUUID(), payload)
	err := publisher.Publish("safebucket", msg)

	if err != nil {
		zap.L().Fatal("Failed to publish event", zap.Error(err))
	}
}
