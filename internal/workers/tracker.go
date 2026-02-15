package workers

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// WorkerTask represents a named operation to be executed during a worker run.
type WorkerTask struct {
	Name string
	Fn   func(ctx context.Context) (int, error)
}

func executeTasks(ctx context.Context, tasks []WorkerTask) []int {
	counts := make([]int, len(tasks))

	for i, task := range tasks {
		count, taskErr := task.Fn(ctx)
		if taskErr != nil {
			zap.L().Error("Cleanup task failed",
				zap.String("task", task.Name),
				zap.Error(taskErr))
		}
		counts[i] = count
	}

	return counts
}

// StartPeriodicWorker runs an immediate cleanup cycle, then repeats on interval.
func StartPeriodicWorker(ctx context.Context, workerName string, interval time.Duration, tasks []WorkerTask) {
	zap.L().Info("Starting worker",
		zap.String("worker", workerName),
		zap.Duration("interval", interval))

	runWorkerCycle(ctx, workerName, tasks)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Worker shutting down", zap.String("worker", workerName))
			return
		case <-ticker.C:
			runWorkerCycle(ctx, workerName, tasks)
		}
	}
}

// runWorkerCycle executes a single tracked worker cycle, logging timing and per-task counts.
func runWorkerCycle(ctx context.Context, workerName string, tasks []WorkerTask) {
	startTime := time.Now()
	zap.L().Info("Starting worker cycle", zap.String("worker", workerName))

	counts := executeTasks(ctx, tasks)

	fields := []zap.Field{zap.String("worker", workerName)}
	for i, task := range tasks {
		fields = append(fields, zap.Int(task.Name, counts[i]))
	}
	fields = append(fields, zap.Duration("duration", time.Since(startTime)))

	zap.L().Info("Worker cycle complete", fields...)
}
