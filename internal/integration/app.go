//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/core"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testJWTSecret        = "integration-test-jwt-secret"
	testMFAEncryptionKey = "01234567890123456789012345678901"
	testAdminEmail       = "admin@safebucket.test"
	testAdminPassword    = "admin-correct-horse-staple"
	testPassword         = "correct-horse-battery-staple"
)

var testProvider DBProvider

type TestApp struct {
	BaseURL     string
	Config      models.Configuration
	DB          *gorm.DB
	Cache       cache.ICache
	Storage     storage.IStorage
	Publisher   messaging.IPublisher
	Notifier    notifier.INotifier
	Activity    activity.IActivityLogger
	NotifyDir   string
	ActivityDir string

	server *httptest.Server
	client *http.Client
}

func DefaultTestConfig(t *testing.T) models.Configuration {
	t.Helper()

	notifyDir := filepath.Join(t.TempDir(), "notifications")
	require.NoError(t, os.MkdirAll(notifyDir, 0o750))

	activityDir := filepath.Join(t.TempDir(), "activity")
	require.NoError(t, os.MkdirAll(activityDir, 0o750))

	return models.Configuration{
		App: models.AppConfiguration{
			Profile:                          configuration.ProfileDefault,
			AdminEmail:                       testAdminEmail,
			AdminPassword:                    testAdminPassword,
			APIURL:                           "http://api.test",
			AllowedOrigins:                   []string{"*"},
			JWTSecret:                        testJWTSecret,
			MFAEncryptionKey:                 testMFAEncryptionKey,
			MFARequired:                      false,
			AccessTokenExpiry:                60,
			RefreshTokenExpiry:               600,
			MFATokenExpiry:                   5,
			LogLevel:                         "info",
			Port:                             8080,
			StaticFiles:                      models.StaticConfiguration{Enabled: false},
			TrustedProxies:                   []string{"127.0.0.1"},
			WebURL:                           "http://web.test",
			TrashRetentionDays:               7,
			MaxUploadSize:                    32 << 20,
			AuthenticatedRequestsPerMinute:   10_000,
			UnauthenticatedRequestsPerMinute: 10_000,
		},
		Auth: models.AuthConfiguration{
			Providers: map[string]models.ProviderConfiguration{
				string(models.LocalProviderType): {
					Name: string(models.LocalProviderType),
					Type: models.LocalProviderType,
				},
			},
		},
		Cache: models.CacheConfiguration{Type: "memory"},
		Events: models.EventsConfiguration{
			Type: configuration.ProviderMemory,
			Queues: map[string]models.QueueConfig{
				configuration.EventsNotifications:  {Name: "test-" + configuration.EventsNotifications},
				configuration.EventsObjectDeletion: {Name: "test-" + configuration.EventsObjectDeletion},
				configuration.EventsBucketEvents:   {Name: "test-" + configuration.EventsBucketEvents},
			},
		},
		Notifier: models.NotifierConfiguration{
			Type:       "filesystem",
			Filesystem: &models.FilesystemNotifierConfiguration{Directory: notifyDir},
		},
		Activity: models.ActivityConfiguration{
			Type:       "filesystem",
			Filesystem: &models.FilesystemActivityConfiguration{Directory: activityDir},
		},
	}
}

func BootTestApp(t *testing.T, cfg models.Configuration) *TestApp {
	t.Helper()

	db := testProvider.Setup(t)

	minioInstance := StartMinIO(t)
	cfg.Storage = models.StorageConfiguration{
		Type: configuration.ProviderMinio,
		Minio: &models.MinioStorageConfiguration{
			BucketName:       minioInstance.Bucket,
			Endpoint:         minioInstance.Endpoint,
			ExternalEndpoint: minioInstance.ExternalEndpoint,
			ClientID:         minioInstance.AccessKey,
			ClientSecret:     minioInstance.SecretKey,
			Region:           "us-east-1",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := core.Boot(ctx, cfg, core.BootOptions{DB: db})
	t.Cleanup(func() { app.Cache.Close() })
	t.Cleanup(func() { _ = app.ActivityLogger.Close() })

	server := httptest.NewServer(app.Router)

	t.Cleanup(func() {
		server.Close()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Logf("integration: app shutdown: %v", err)
		}

		cancel()
	})

	notifyDir := ""
	if cfg.Notifier.Filesystem != nil {
		notifyDir = cfg.Notifier.Filesystem.Directory
	}
	activityDir := ""
	if cfg.Activity.Filesystem != nil {
		activityDir = cfg.Activity.Filesystem.Directory
	}

	return &TestApp{
		BaseURL:     server.URL,
		Config:      cfg,
		DB:          app.DB,
		Cache:       app.Cache,
		Storage:     app.Storage,
		Publisher:   app.EventRouter,
		Notifier:    app.Notifier,
		Activity:    app.ActivityLogger,
		NotifyDir:   notifyDir,
		ActivityDir: activityDir,
		server:      server,
		client:      server.Client(),
	}
}

func (a *TestApp) URL(path string) string {
	return a.BaseURL + path
}

func (a *TestApp) LoginAs(t *testing.T, email string) string {
	t.Helper()

	var resp models.AuthLoginResponse
	status := a.Do(t, http.MethodPost, "/api/v1/auth/login", "", models.AuthLoginBody{
		Email:    email,
		Password: testPassword,
	}, &resp)
	require.Equal(t, http.StatusCreated, status, "login should succeed")
	require.NotEmpty(t, resp.AccessToken)
	return resp.AccessToken
}

func (a *TestApp) Do(
	t *testing.T,
	method, path, token string,
	body, out any,
) int {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, a.URL(path), reqBody)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := a.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if raw, readErr := io.ReadAll(resp.Body); readErr == nil && len(raw) > 0 {
			t.Logf("integration: %s %s -> %d body=%s", method, path, resp.StatusCode, string(raw))
		}
		return resp.StatusCode
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		if decodeErr := json.NewDecoder(resp.Body).Decode(out); decodeErr != nil &&
			!errors.Is(decodeErr, io.EOF) {
			require.NoError(t, decodeErr)
		}
	}
	return resp.StatusCode
}

func (a *TestApp) Eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	assert.Eventually(t, cond, 5*time.Second, 50*time.Millisecond, msg)
}

type Notification struct {
	To           string          `json:"to"`
	Subject      string          `json:"subject"`
	TemplateName string          `json:"template_name"`
	Args         json.RawMessage `json:"args"`
	Timestamp    string          `json:"timestamp"`
}

func (a *TestApp) ReadNotifications(t *testing.T) []Notification {
	t.Helper()

	if a.NotifyDir == "" {
		return nil
	}
	entries, err := os.ReadDir(a.NotifyDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		require.NoError(t, err)
	}

	var out []Notification
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(a.NotifyDir, e.Name()))
		if readErr != nil {
			t.Logf("integration: read notification %s: %v", e.Name(), readErr)
			continue
		}
		var n Notification
		if decodeErr := json.Unmarshal(raw, &n); decodeErr != nil {
			t.Logf("integration: decode notification %s: %v", e.Name(), decodeErr)
			continue
		}
		out = append(out, n)
	}
	return out
}
