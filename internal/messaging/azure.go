package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const (
	azureDequeueBatchSize         = 1
	azureVisibilityTimeoutSeconds = 60
	azureMaxDeliveryCount         = 5
	azurePollInterval             = 2 * time.Second
)

type queueEnvelope struct {
	UUID     string            `json:"uuid"`
	Metadata map[string]string `json:"metadata"`
	Payload  []byte            `json:"payload"`
}

type AzurePublisher struct {
	queueName string
	client    *azqueue.QueueClient
}

func NewAzurePublisher(config *models.AzureEventsConfiguration, queueName string) IPublisher {
	return &AzurePublisher{queueName: queueName, client: newAzureQueueClient(config, queueName)}
}

func (p *AzurePublisher) Publish(messages ...*message.Message) error {
	for _, msg := range messages {
		content, err := json.Marshal(queueEnvelope{
			UUID:     msg.UUID,
			Metadata: map[string]string(msg.Metadata),
			Payload:  msg.Payload,
		})
		if err != nil {
			return err
		}

		if _, pubErr := p.client.EnqueueMessage(context.Background(), string(content), nil); pubErr != nil {
			return pubErr
		}
	}

	return nil
}

func (p *AzurePublisher) Close() error {
	return nil
}

type AzureSubscriber struct {
	queueName string
	client    *azqueue.QueueClient
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewAzureSubscriber(config *models.AzureEventsConfiguration, queueName string) ISubscriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &AzureSubscriber{
		queueName: queueName,
		client:    newAzureQueueClient(config, queueName),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *AzureSubscriber) Subscribe() <-chan *message.Message {
	out := make(chan *message.Message)
	go s.poll(out)
	return out
}

func (s *AzureSubscriber) Close() error {
	s.cancel()
	return nil
}

func (s *AzureSubscriber) poll(out chan<- *message.Message) {
	defer close(out)

	for {
		if s.ctx.Err() != nil {
			return
		}

		resp, err := s.client.DequeueMessages(s.ctx, &azqueue.DequeueMessagesOptions{
			NumberOfMessages:  to.Ptr(int32(azureDequeueBatchSize)),
			VisibilityTimeout: to.Ptr(int32(azureVisibilityTimeoutSeconds)),
		})
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			zap.L().Error("Failed to dequeue messages", zap.String("queue", s.queueName), zap.Error(err))
			s.wait(azurePollInterval)
			continue
		}

		if len(resp.Messages) == 0 {
			s.wait(azurePollInterval)
			continue
		}

		for _, queueMessage := range resp.Messages {
			if !s.dispatch(out, queueMessage) {
				return
			}
		}
	}
}

func (s *AzureSubscriber) dispatch(out chan<- *message.Message, queueMessage *azqueue.DequeuedMessage) bool {
	if queueMessage.DequeueCount != nil && *queueMessage.DequeueCount > azureMaxDeliveryCount {
		zap.L().Warn("Dropping poison message after too many deliveries",
			zap.String("queue", s.queueName),
			zap.Int64("dequeue_count", *queueMessage.DequeueCount))
		s.deleteMessage(queueMessage)
		return true
	}

	msg := buildWatermillMessage(azureMessageBody(queueMessage))
	msg.SetContext(s.ctx)

	select {
	case out <- msg:
	case <-s.ctx.Done():
		return false
	}

	select {
	case <-msg.Acked():
		s.deleteMessage(queueMessage)
	case <-msg.Nacked():
	case <-s.ctx.Done():
		return false
	}

	return true
}

func (s *AzureSubscriber) deleteMessage(queueMessage *azqueue.DequeuedMessage) {
	if queueMessage.MessageID == nil || queueMessage.PopReceipt == nil {
		return
	}

	_, err := s.client.DeleteMessage(s.ctx, *queueMessage.MessageID, *queueMessage.PopReceipt, nil)
	if err != nil {
		zap.L().Error("Failed to delete queue message", zap.String("queue", s.queueName), zap.Error(err))
	}
}

func (s *AzureSubscriber) wait(d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-s.ctx.Done():
	case <-timer.C:
	}
}

func newAzureQueueClient(config *models.AzureEventsConfiguration, queueName string) *azqueue.QueueClient {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		zap.L().Fatal("Failed to create Azure credential", zap.Error(err))
	}

	queueURL := fmt.Sprintf("https://%s.queue.core.windows.net/%s", config.AccountName, queueName)
	client, err := azqueue.NewQueueClient(queueURL, cred, nil)
	if err != nil {
		zap.L().Fatal("Failed to create Azure queue client", zap.Error(err))
	}

	return client
}

func azureMessageBody(queueMessage *azqueue.DequeuedMessage) []byte {
	if queueMessage.MessageText == nil {
		return nil
	}

	text := *queueMessage.MessageText
	if decoded, err := base64.StdEncoding.DecodeString(text); err == nil && json.Valid(decoded) {
		return decoded
	}

	return []byte(text)
}

func buildWatermillMessage(body []byte) *message.Message {
	var envelope queueEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Payload) > 0 {
		uid := envelope.UUID
		if uid == "" {
			uid = watermill.NewUUID()
		}

		msg := message.NewMessage(uid, envelope.Payload)
		for key, value := range envelope.Metadata {
			msg.Metadata.Set(key, value)
		}
		return msg
	}

	return message.NewMessage(watermill.NewUUID(), body)
}
