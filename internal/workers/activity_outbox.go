package workers

import (
	"context"
	"encoding/json"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/models"

	"gorm.io/gorm"
)

type ActivityOutboxWorker struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

func (w *ActivityOutboxWorker) Start(ctx context.Context) {
	StartPeriodicWorker(ctx, "activity_outbox", outboxPollInterval, []WorkerTask{
		{Name: "messages", Fn: func(taskCtx context.Context) (int, error) {
			return processOutboxMessages(
				taskCtx,
				w.DB,
				"activity_outbox_messages",
				"activity outbox",
				w.deliver,
			)
		}},
	})
}

func (w *ActivityOutboxWorker) deliver(outboxMessage *pendingOutboxMessage) error {
	var activityMessage models.Activity
	if err := json.Unmarshal([]byte(outboxMessage.Payload), &activityMessage); err != nil {
		return err
	}
	activityMessage.ID = outboxMessage.ID.String()
	return w.ActivityLogger.Send(activityMessage)
}
