package core

import (
	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/models"
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
