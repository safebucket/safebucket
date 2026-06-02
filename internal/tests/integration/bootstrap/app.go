//go:build integration

package bootstrap

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type OIDCSetup struct {
	ProviderKey string
	Users       []DexUser
}

const TestPassword = "correct-horse-battery-staple"

//go:embed scenarios/*.yaml
var scenariosFS embed.FS

func ActiveScenarios() []string {
	if s := os.Getenv("TEST_SCENARIO"); s != "" {
		return []string{s}
	}

	entries, err := scenariosFS.ReadDir("scenarios")
	if err != nil {
		panic("integration: failed to read embedded scenarios: " + err.Error())
	}

	if len(entries) == 0 {
		panic("integration: no scenarios found in embedded scenarios/")
	}

	scenarios := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) != ".yaml" {
			continue
		}
		scenarios = append(scenarios, name[:len(name)-5])
	}
	return scenarios
}

type TestApp struct {
	BaseURL     string
	Config      models.Configuration
	Cache       cache.ICache
	Storage     storage.IStorage
	Publisher   messaging.IPublisher
	Notifier    notifier.INotifier
	Activity    activity.IActivityLogger
	NotifyDir   string
	ActivityDir string

	db               *gorm.DB
	cachedAdminToken string
	server           *httptest.Server
	client           *http.Client
}

func LoadScenario(t *testing.T, name string) models.Configuration {
	t.Helper()

	raw, err := scenariosFS.ReadFile(filepath.Join("scenarios", name+".yaml"))
	require.NoError(t, err, "read embedded scenario %s", name)

	cfg, err := configuration.Load(configuration.LoadOptions{
		ConfigBytes: raw,
		SkipEnv:     true,
	})
	require.NoError(t, err, "load scenario %s", name)
	return cfg
}

func BootScenario(t *testing.T, name string) *TestApp {
	return BootTestApp(t, LoadScenario(t, name))
}

func BootTestApp(t *testing.T, cfg models.Configuration) *TestApp {
	t.Helper()
	return bootTestApp(t, cfg, nil)
}

func BootScenarioWithOIDC(t *testing.T, name string, setup OIDCSetup) *TestApp {
	t.Helper()
	require.NotEmpty(t, setup.ProviderKey, "OIDCSetup.ProviderKey is required")
	return bootTestApp(t, LoadScenario(t, name), &setup)
}

func bootTestApp(t *testing.T, cfg models.Configuration, oidc *OIDCSetup) *TestApp {
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

	server := httptest.NewUnstartedServer(nil)
	t.Cleanup(func() { server.Close() })
	baseURL := "http://" + server.Listener.Addr().String()

	if oidc != nil {
		cfg.App.APIURL = baseURL
		callbackURL := fmt.Sprintf("%s/api/v1/auth/providers/%s/callback", baseURL, oidc.ProviderKey)
		dex := StartDex(t, callbackURL, oidc.Users)

		if cfg.Auth.Providers == nil {
			cfg.Auth.Providers = map[string]models.ProviderConfiguration{}
		}
		cfg.Auth.Providers[oidc.ProviderKey] = models.ProviderConfiguration{
			Name: "Dex",
			Type: models.OIDCProviderType,
			OIDC: models.OIDCConfiguration{
				ClientID:     dex.ClientID,
				ClientSecret: dex.ClientSecret,
				Issuer:       dex.Issuer,
			},
		}
	}

	require.NoError(t, configuration.Validate(cfg), "final scenario config failed validation")
	middlewares.InitValidator(cfg.App.MaxUploadSize)

	ctx, cancel := context.WithCancel(context.Background())

	app := core.Boot(ctx, cfg, core.BootOptions{DB: db})
	t.Cleanup(func() { app.Cache.Close() })
	t.Cleanup(func() { _ = app.ActivityLogger.Close() })

	server.Config.Handler = app.Router
	server.Start()

	t.Cleanup(func() {
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
		db:          app.DB,
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

func (a *TestApp) DB() *gorm.DB {
	return a.db
}

func (a *TestApp) URL(path string) string {
	return a.BaseURL + path
}

func (a *TestApp) LoginAs(t *testing.T, email string) string {
	t.Helper()
	status, token := a.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/login", "",
		models.AuthLoginBody{Email: email, Password: TestPassword})
	require.Equal(t, http.StatusOK, status, "login should succeed")
	require.NotEmpty(t, token, "access token cookie should be set after login")
	return token
}

func (a *TestApp) LoginRefreshToken(t *testing.T, email string) string {
	t.Helper()

	body, err := json.Marshal(models.AuthLoginBody{Email: email, Password: TestPassword})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		a.URL("/api/v1/auth/login"), bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	require.NoError(t, err)
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode, "login should succeed")

	for _, c := range resp.Cookies() {
		if c.Name == "safebucket_refresh_token" {
			return c.Value
		}
	}
	t.Fatal("refresh token cookie not set after login")
	return ""
}

func (a *TestApp) LoginAdmin(t *testing.T) string {
	t.Helper()
	status, token := a.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/login", "",
		models.AuthLoginBody{Email: a.Config.App.AdminEmail, Password: a.Config.App.AdminPassword})
	require.Equal(t, http.StatusOK, status, "admin login should succeed")
	require.NotEmpty(t, token, "access token cookie should be set after admin login")
	return token
}

func (a *TestApp) DoGetAuthCookie(t *testing.T, method, path, token string, body any) (int, string) {
	t.Helper()
	return a.doGetCookie(t, method, path, token, body, "safebucket_access_token")
}

func (a *TestApp) DoGetMFACookie(t *testing.T, method, path, token string, body any) (int, string) {
	t.Helper()
	return a.doGetCookie(t, method, path, token, body, "safebucket_mfa_token")
}

func (a *TestApp) doGetCookie(
	t *testing.T, method, path, token string, body any, cookieName string,
) (int, string) {
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

	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			return resp.StatusCode, c.Value
		}
	}
	return resp.StatusCode, ""
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
		if raw, readErr := io.ReadAll(resp.Body); readErr == nil && len(raw) > 0 && integrationVerbose() {
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

func (a *TestApp) DoStatus(t *testing.T, method, path, token string, body any) int {
	t.Helper()
	return a.Do(t, method, path, token, body, nil)
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

func (a *TestApp) adminToken(t *testing.T) string {
	t.Helper()
	if a.cachedAdminToken == "" {
		a.cachedAdminToken = a.LoginAdmin(t)
	}
	return a.cachedAdminToken
}

func (a *TestApp) CreateUser(t *testing.T, email string) models.User {
	t.Helper()
	var user models.User
	status := a.Do(t, http.MethodPost, "/api/v1/users", a.adminToken(t),
		models.UserCreateBody{FirstName: "Test", LastName: "User", Email: email, Password: TestPassword},
		&user)
	require.Equal(t, http.StatusCreated, status, "create user %s", email)
	return user
}

func (a *TestApp) SetUserRole(t *testing.T, email string, role models.Role) {
	t.Helper()
	require.NoError(t, a.db.Model(&models.User{}).Where("email = ?", email).Update("role", role).Error)
}

func (a *TestApp) CreateBucket(t *testing.T, token, name string) models.Bucket {
	t.Helper()
	var bucket models.Bucket
	status := a.Do(t, http.MethodPost, "/api/v1/buckets", token,
		models.BucketCreateUpdateBody{Name: name}, &bucket)
	require.Equal(t, http.StatusCreated, status, "create bucket %s", name)
	return bucket
}

func (a *TestApp) AddMembers(t *testing.T, token, bucketID string, members []models.BucketMemberBody) {
	t.Helper()
	status := a.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketID), token,
		models.UpdateMembersBody{Members: members})
	require.Equal(t, http.StatusNoContent, status, "set members for bucket %s", bucketID)
}

func (a *TestApp) GetMembers(t *testing.T, token, bucketID string) []models.BucketMember {
	t.Helper()
	var page models.Page[models.BucketMember]
	status := a.Do(t, http.MethodGet, fmt.Sprintf("/api/v1/buckets/%s/members", bucketID), token, nil, &page)
	require.Equal(t, http.StatusOK, status, "get members for bucket %s", bucketID)
	return page.Data
}

func (a *TestApp) BackdateShareExpiry(t *testing.T, shareID string, when time.Time) {
	t.Helper()
	require.NoError(t,
		a.db.Exec("UPDATE shares SET expires_at = ? WHERE id = ?", when.UTC(), shareID).Error,
	)
}

func (a *TestApp) BackdateFileExpiry(t *testing.T, fileID string, when time.Time) {
	t.Helper()
	require.NoError(t,
		a.db.Exec("UPDATE files SET expires_at = ? WHERE id = ?", when.UTC(), fileID).Error,
	)
}

func (a *TestApp) TrashFile(t *testing.T, token, bucketID, fileID string) {
	t.Helper()
	status := a.DoStatus(t, http.MethodPatch,
		fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, fileID), token,
		models.FilePatchBody{Status: string(models.FileStatusDeleted)})
	require.Equal(t, http.StatusNoContent, status, "trash file %s", fileID)
}

func (a *TestApp) CreateFolder(t *testing.T, token, bucketID, name string) models.Folder {
	t.Helper()
	var folder models.Folder
	status := a.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/folders", bucketID), token,
		models.FolderCreateBody{Name: name}, &folder)
	require.Equal(t, http.StatusCreated, status, "create folder %s", name)
	return folder
}

func (a *TestApp) CreateShare(
	t *testing.T,
	token, bucketID string,
	body models.ShareCreateBody,
) models.Share {
	t.Helper()
	var share models.Share
	status := a.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/shares", bucketID), token,
		body, &share)
	require.Equal(t, http.StatusCreated, status, "create share %s", body.Name)
	return share
}

func (a *TestApp) AuthenticateShare(t *testing.T, shareID, password string) (int, string) {
	t.Helper()
	return a.doGetCookie(t, http.MethodPost,
		fmt.Sprintf("/api/v1/shares/%s/auth", shareID),
		"",
		models.ShareAuthBody{Password: password},
		configuration.CookieShareToken,
	)
}

func (a *TestApp) DoPublicShare(
	t *testing.T,
	method, path, shareCookie string,
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
	if shareCookie != "" {
		cookie := &http.Cookie{
			Name:  configuration.CookieShareToken,
			Value: shareCookie,
		}
		req.AddCookie(cookie)
	}

	resp, err := a.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if raw, readErr := io.ReadAll(resp.Body); readErr == nil && len(raw) > 0 && integrationVerbose() {
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

func (a *TestApp) UploadTestFile(t *testing.T, token, bucketID, name string) string {
	t.Helper()
	return a.UploadFileInto(t, token, bucketID, nil, name)
}

func (a *TestApp) UploadFileInto(
	t *testing.T,
	token, bucketID string,
	folderID *uuid.UUID,
	name string,
) string {
	t.Helper()

	var transfer models.FileTransferResponse
	status := a.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
		models.FileTransferBody{Name: name, Size: 5, FolderID: folderID}, &transfer)
	require.Equal(t, http.StatusCreated, status, "create upload slot for %s", name)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range transfer.Body {
		require.NoError(t, mw.WriteField(k, v))
	}
	fw, err := mw.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = fw.Write([]byte("test!"))
	require.NoError(t, err)
	mw.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, transfer.URL, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	uploadResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	uploadResp.Body.Close()
	require.Less(t, uploadResp.StatusCode, 300, "MinIO presigned upload should succeed, got %d", uploadResp.StatusCode)

	require.Equal(t, http.StatusNoContent,
		a.DoStatus(t, http.MethodPatch, fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID), token,
			models.FilePatchBody{Status: string(models.FileStatusUploaded)}),
		"confirm upload for %s", name)

	return transfer.ID
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
