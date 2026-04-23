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
	"github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testPassword = "correct-horse-battery-staple"

func ActiveScenarios() []string {
	if s := os.Getenv("TEST_SCENARIO"); s != "" {
		return []string{s}
	}

	matches, err := filepath.Glob(filepath.Join("scenarios", "*.yaml"))
	if err != nil {
		panic("integration: failed to glob test scenarios: " + err.Error())
	}

	if len(matches) == 0 {
		panic("integration: no scenarios found in scenarios/")
	}

	scenarios := make([]string, 0, len(matches))
	for _, m := range matches {
		name := filepath.Base(m)
		scenarios = append(scenarios, name[:len(name)-5]) // strip .yaml
	}
	return scenarios
}

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

func LoadScenario(t *testing.T, name string) models.Configuration {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("scenarios", name+".yaml"))
	require.NoError(t, err, "resolve scenario path")

	cfg, err := configuration.Load(configuration.LoadOptions{
		ConfigFilePath: path,
		SkipEnv:        true,
	})
	require.NoError(t, err, "load scenario %s", name)
	return cfg
}

func BootScenario(t *testing.T, name string) *TestApp {
	return BootTestApp(t, LoadScenario(t, name))
}

func BootTestApp(t *testing.T, cfg models.Configuration) *TestApp {
	t.Helper()

	provider := providerForDialect(t, cfg.Database.Type)
	db := provider.Setup(t)
	injectDatabaseConfig(t, &cfg, provider, db)

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

	notifyDir := filepath.Join(t.TempDir(), "notifications")
	require.NoError(t, os.MkdirAll(notifyDir, 0o750))
	cfg.Notifier.Filesystem = &models.FilesystemNotifierConfiguration{Directory: notifyDir}

	activityDir := filepath.Join(t.TempDir(), "activity")
	require.NoError(t, os.MkdirAll(activityDir, 0o750))
	cfg.Activity.Filesystem = &models.FilesystemActivityConfiguration{Directory: activityDir}

	require.NoError(t, configuration.Validate(cfg), "final scenario config failed validation")
	middlewares.InitValidator(cfg.App.MaxUploadSize)

	ctx, cancel := context.WithCancel(context.Background())

	app := core.Boot(ctx, cfg, core.BootOptions{DB: db})
	t.Cleanup(func() { app.Cache.Close() })
	t.Cleanup(func() { _ = app.ActivityLogger.Close() })

	server := httptest.NewServer(app.Router)

	t.Cleanup(func() {
		server.Close()

		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Logf("integration: app shutdown: %v", err)
		}
	})

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

func providerForDialect(t *testing.T, dialect string) DBProvider {
	t.Helper()
	switch dialect {
	case configuration.ProviderPostgres:
		return &PostgresProvider{}
	case configuration.ProviderSQLite:
		return &SQLiteProvider{}
	default:
		require.Failf(t, "unsupported dialect", "scenario requested dialect %q", dialect)
		return nil
	}
}

func injectDatabaseConfig(t *testing.T, cfg *models.Configuration, provider DBProvider, db *gorm.DB) {
	t.Helper()
	switch provider.Dialect() {
	case configuration.ProviderPostgres:
		pg, ok := provider.(*PostgresProvider)
		require.True(t, ok, "provider must be *PostgresProvider for postgres dialect")
		cfg.Database.Postgres = pg.ConfigFor(t)
	case configuration.ProviderSQLite:
		if cfg.Database.SQLite == nil {
			cfg.Database.SQLite = &models.SQLiteDatabaseConfig{Path: ":memory:"}
		}
	default:
		require.Failf(t, "unsupported dialect", "provider dialect %q", provider.Dialect())
	}
}
