package core

import (
	"api/internal/models"
	"api/internal/notifier"
)

func NewNotifier(config models.NotifierConfiguration) notifier.INotifier {
	switch config.Type {
	case "smtp":
		return notifier.NewSMTPNotifier(*config.SMTP)
	default:
		return nil
	}
}
