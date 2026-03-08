package core

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
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
