package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkerRunStatus string

const (
	WorkerRunStatusRunning   WorkerRunStatus = "running"
	WorkerRunStatusCompleted WorkerRunStatus = "completed"
	WorkerRunStatusFailed    WorkerRunStatus = "failed"
)

type WorkerRun struct {
	ID         uuid.UUID       `gorm:"type:uuid;primarykey;default:gen_random_uuid()"    json:"id"`
	WorkerName string          `gorm:"type:varchar(64);not null"                         json:"worker_name"`
	Status     WorkerRunStatus `gorm:"type:worker_run_status;not null;default:'running'" json:"status"`
	StartedAt  time.Time       `gorm:"not null;default:now()"                            json:"started_at"`
	EndedAt    time.Time       `                                                         json:"ended_at"`
}
