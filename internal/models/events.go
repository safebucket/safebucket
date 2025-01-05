package models

import "time"

type Event struct {
	EventID    string            `json:"event_id"`
	Type       string            `json:"type"`
	OccurredAt time.Time         `json:"occurred_at"`
	Actor      Actor             `json:"actor"`
	Source     string            `json:"source"`
	Payload    interface{}       `json:"payload"`
	Metadata   map[string]string `json:"metadata"`
}

type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
