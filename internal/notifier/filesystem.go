package notifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"api/internal/models"

	"go.uber.org/zap"
)

type FilesystemNotifier struct {
	directory string
}

func NewFilesystemNotifier(config models.FilesystemNotifierConfiguration) *FilesystemNotifier {
	if err := os.MkdirAll(config.Directory, 0750); err != nil {
		zap.L().Fatal("Failed to create notification directory", zap.Error(err))
	}
	return &FilesystemNotifier{directory: config.Directory}
}

func (f *FilesystemNotifier) NotifyFromTemplate(
	to string,
	subject string,
	templateName string,
	data any,
) error {
	entry := map[string]any{
		"to":            to,
		"subject":       subject,
		"template_name": templateName,
		"args":          data,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	content, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	filename := fmt.Sprintf("%d.json", time.Now().UnixNano())
	path := filepath.Join(f.directory, filename)

	if err = os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("failed to write notification file: %w", err)
	}

	zap.L().Info("Notification written to filesystem",
		zap.String("path", path),
		zap.String("to", to),
		zap.String("subject", subject),
	)

	return nil
}
