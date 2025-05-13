package core

import (
	"api/internal/logs"
	"api/internal/models"
)

func NewLogPublisher(config models.LogConfiguration) logs.ILogClient {
	switch config.Type {
	case "loki":
		return logs.NewLokiClient(config)
	default:
		return nil
	}
}
