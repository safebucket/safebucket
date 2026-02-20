package core

import (
	"api/internal/models"
	"api/internal/notifier"
)

func NewNotifier(config models.NotifierConfiguration) notifier.INotifier {
	switch config.Type {
	case "smtp":
		return notifier.NewSMTPNotifier(*config.SMTP)
	case "filesystem":
		return notifier.NewFilesystemNotifier(*config.Filesystem)
	default:
		return nil
	}
}
