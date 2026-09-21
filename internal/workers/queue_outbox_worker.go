package workers

import (
	"context"

	"github.com/safebucket/safebucket/internal/messaging"

	"github.com/ThreeDotsLabs/watermill/message"
	"gorm.io/gorm"
)

type QueueOutboxWorker struct {
	DB        *gorm.DB
	Publisher messaging.IPublisher
}

func (w *QueueOutboxWorker) Start(ctx context.Context) {
	StartPeriodicWorker(ctx, "queue_outbox", outboxPollInterval, []WorkerTask{
		{Name: "messages", Fn: func(taskCtx context.Context) (int, error) {
			return processOutboxMessages(taskCtx, w.DB, "queue_outbox_messages", "queue outbox", w.publish)
		}},
	})
}

func (w *QueueOutboxWorker) publish(outboxMessage *pendingOutboxMessage) error {
	msg := message.NewMessage(outboxMessage.ID.String(), []byte(outboxMessage.Payload))
	msg.Metadata.Set("type", outboxMessage.MessageType)
	return w.Publisher.Publish(msg)
}
