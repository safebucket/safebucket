package workers

import (
	"fmt"
	"time"

	"api/internal/activity"
	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunTracker records worker run status in the database and pushes details to Loki.
type RunTracker struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

// StartRun creates a new WorkerRun record with status "running".
func (t *RunTracker) StartRun(workerName string) (*models.WorkerRun, error) {
	run := models.WorkerRun{
		WorkerName: workerName,
		Status:     models.WorkerRunStatusRunning,
		StartedAt:  time.Now(),
	}

	if err := t.DB.Create(&run).Error; err != nil {
		return nil, fmt.Errorf("failed to create worker run: %w", err)
	}

	return &run, nil
}

// CompleteRun marks a run as completed and pushes run details to Loki.
func (t *RunTracker) CompleteRun(run *models.WorkerRun) {
	now := time.Now()

	if err := t.DB.Model(run).Updates(map[string]interface{}{
		"status":   models.WorkerRunStatusCompleted,
		"ended_at": now,
	}).Error; err != nil {
		zap.L().Error("Failed to mark worker run as completed",
			zap.String("run_id", run.ID.String()),
			zap.Error(err))
	}
}

// FailRun marks a run as failed, records the error, and pushes it to Loki.
func (t *RunTracker) FailRun(run *models.WorkerRun) {
	now := time.Now()

	if dbErr := t.DB.Model(run).Updates(map[string]interface{}{
		"status":   models.WorkerRunStatusFailed,
		"ended_at": now,
	}).Error; dbErr != nil {
		zap.L().Error("Failed to mark worker run as failed",
			zap.String("run_id", run.ID.String()),
			zap.Error(dbErr))
	}
}
