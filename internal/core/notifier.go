package core

import (
	"api/internal/models"
	"api/internal/notifier"
)

func NewNotifier(config models.NotifierConfiguration) notifier.INotifier {
	switch config.Type {
	case "smtp":
		return notifier.NewSMTPNotifier(models.MailerConfiguration(*config.SMTP))
	default:
		// Fall back to SMTP if type is not specified or invalid
		return notifier.NewSMTPNotifier(models.MailerConfiguration(*config.SMTP))
	}
}
