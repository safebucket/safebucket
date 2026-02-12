package workers

import (
	"context"
	"fmt"
	"time"

	"api/internal/activity"
	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkerTask represents a named operation to be executed during a worker run.
type WorkerTask struct {
	Name string
	Fn   func(ctx context.Context) (int, error)
}

func (t *RunTracker) ExecuteTasks(
	ctx context.Context,
	workerName string,
	tasks []WorkerTask,
) []int {
	run, err := t.StartRun(workerName)
	if err != nil {
		zap.L().Error("Failed to start worker run tracking", zap.Error(err))
		return make([]int, len(tasks))
	}

	var runFailed bool
	counts := make([]int, len(tasks))

	for i, task := range tasks {
		count, taskErr := task.Fn(ctx)
		if taskErr != nil {
			zap.L().Error("Cleanup task failed",
				zap.String("task", task.Name),
				zap.Error(taskErr))
			runFailed = true
		}
		counts[i] = count
	}

	if runFailed {
		t.FailRun(run)
	} else {
		t.CompleteRun(run)
	}

	return counts
}

// StartPeriodicWorker runs an immediate cleanup cycle, then repeats on interval
func StartPeriodicWorker(ctx context.Context, tracker *RunTracker, workerName string, interval time.Duration, tasks []WorkerTask) {
	zap.L().Info("Starting worker",
		zap.String("worker", workerName),
		zap.Duration("interval", interval))

	runWorkerCycle(ctx, tracker, workerName, tasks)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Worker shutting down", zap.String("worker", workerName))
			return
		case <-ticker.C:
			runWorkerCycle(ctx, tracker, workerName, tasks)
		}
	}
}

// runWorkerCycle executes a single tracked worker cycle, logging timing and per-task counts.
func runWorkerCycle(ctx context.Context, tracker *RunTracker, workerName string, tasks []WorkerTask) {
	startTime := time.Now()
	zap.L().Info("Starting worker cycle", zap.String("worker", workerName))

	counts := tracker.ExecuteTasks(ctx, workerName, tasks)

	fields := []zap.Field{zap.String("worker", workerName)}
	for i, task := range tasks {
		fields = append(fields, zap.Int(task.Name, counts[i]))
	}
	fields = append(fields, zap.Duration("duration", time.Since(startTime)))

	zap.L().Info("Worker cycle complete", fields...)
}

type RunTracker struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

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
