package models

import "github.com/google/uuid"

const (
	MoveStatusOK    = "ok"
	MoveStatusError = "error"
)

type MoveResult struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
	Code   string    `json:"code,omitempty"`
}

type MoveResponse struct {
	Results []MoveResult `json:"results"`
}
