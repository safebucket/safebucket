package notifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"api/internal/models"
)

func newTestFilesystemNotifier(t *testing.T) (*FilesystemNotifier, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "notifications")
	config := models.FilesystemNotifierConfiguration{
		Directory: dir,
	}
	n := NewFilesystemNotifier(config)
	return n, dir
}

func TestFilesystemNotifyFromTemplate_WritesFile(t *testing.T) {
	n, dir := newTestFilesystemNotifier(t)

	data := map[string]string{
		"WebURL":       "http://localhost:3000",
		"Secret":       "123456",
		"ChallengeURL": "http://localhost:3000/reset",
	}

	err := n.NotifyFromTemplate("user@example.com", "Reset your password", "password_reset", data)
	if err != nil {
		t.Fatalf("NotifyFromTemplate failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var result map[string]any
	if err = json.Unmarshal(content, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["to"] != "user@example.com" {
		t.Errorf("expected to=user@example.com, got %v", result["to"])
	}
	if result["subject"] != "Reset your password" {
		t.Errorf("expected subject='Reset your password', got %v", result["subject"])
	}
	if result["template_name"] != "password_reset" {
		t.Errorf("expected template_name=password_reset, got %v", result["template_name"])
	}
	if result["args"] == nil {
		t.Error("expected non-nil args")
	}
	if result["timestamp"] == nil || result["timestamp"] == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestFilesystemNotifyFromTemplate_MultipleNotifications(t *testing.T) {
	n, dir := newTestFilesystemNotifier(t)

	data := map[string]string{
		"WebURL":       "http://localhost:3000",
		"Secret":       "123456",
		"ChallengeURL": "http://localhost:3000/reset",
	}

	for i := range 3 {
		err := n.NotifyFromTemplate("user@example.com", "Reset your password", "password_reset", data)
		if err != nil {
			t.Fatalf("NotifyFromTemplate call %d failed: %v", i, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 files, got %d", len(entries))
	}
}

func TestFilesystemNotifier_DirectoryCreation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep", "notifications")
	config := models.FilesystemNotifierConfiguration{
		Directory: dir,
	}

	_ = NewFilesystemNotifier(config)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}
