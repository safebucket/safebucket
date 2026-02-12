package core

import (
	"api/internal/activity"
	"api/internal/models"
)

func NewActivityLogger(config models.ActivityConfiguration) activity.IActivityLogger {
	switch config.Type {
	case "loki":
		return activity.NewLokiClient(config)
	case "filesystem":
		return activity.NewFilesystemClient(config)
	default:
		return nil
	}
}
