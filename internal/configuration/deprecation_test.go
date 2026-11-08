package configuration

import (
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestMigrateDeprecatedKeys(t *testing.T) {
	t.Run("renamed keys is kept", func(t *testing.T) {
		cfg := loadTestConfig(t, `
database:
  type: postgres
  ssl_mode: disable
app:
  enable_static_files: false
  jwt_secret: old-secret
`)

		assert.Equal(t, "disable", cfg.Database.Postgres.SSLMode)
		assert.False(t, cfg.App.StaticFiles.Enabled)
		assert.Equal(t, "old-secret", cfg.App.TokenSecret)
	})

	t.Run("new key wins over the deprecated one", func(t *testing.T) {
		cfg := loadTestConfig(t, `
database:
  type: postgres
  ssl_mode: disable
  postgres:
    sslmode: require
`)

		assert.Equal(t, "require", cfg.Database.Postgres.SSLMode)
	})
}

func TestMigrateDeprecatedKeys_Logging(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	restoreLogger := zap.ReplaceGlobals(zap.New(core))
	defer restoreLogger()

	cases := []struct {
		name   string
		values map[string]any
		oldKey string
	}{
		{"renamed key", map[string]any{"database.ssl_mode": "disable"}, "database.ssl_mode"},
		{"renamed key shadowed by new key", map[string]any{
			"database.ssl_mode":         "disable",
			"database.postgres.sslmode": "require",
		}, "database.ssl_mode"},
		{"removed key", map[string]any{"database.max_open_conns": 100}, "database.max_open_conns"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs.TakeAll()
			k := koanf.New(".")
			for key, value := range tc.values {
				require.NoError(t, k.Set(key, value))
			}

			migrateDeprecatedKeys(k)

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, zap.WarnLevel, entries[0].Level)
			assert.Equal(t, tc.oldKey, entries[0].ContextMap()["old_key"])
			assert.False(t, k.Exists(tc.oldKey))
		})
	}
}

func TestMigrateGlobalMFARequired(t *testing.T) {
	t.Run("true migrates onto the local provider regardless of its key", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("app.mfa_required", true))
		require.NoError(t, k.Set("auth.providers.passwords.type", "local"))
		require.NoError(t, k.Set("auth.providers.okta.type", "oidc"))

		migrateGlobalMFARequired(k)

		assert.True(t, k.Bool("auth.providers.passwords.mfa_required"))
		assert.False(t, k.Exists("auth.providers.okta.mfa_required"))
		assert.False(t, k.Exists("app.mfa_required"))
	})

	t.Run("false is dropped without touching providers", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("app.mfa_required", false))
		require.NoError(t, k.Set("auth.providers.local.type", "local"))

		migrateGlobalMFARequired(k)

		assert.False(t, k.Exists("auth.providers.local.mfa_required"))
		assert.False(t, k.Exists("app.mfa_required"))
	})

	t.Run("explicit provider value is not overridden", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("app.mfa_required", true))
		require.NoError(t, k.Set("auth.providers.local.type", "local"))
		require.NoError(t, k.Set("auth.providers.local.mfa_required", false))

		migrateGlobalMFARequired(k)

		assert.False(t, k.Bool("auth.providers.local.mfa_required"))
		assert.False(t, k.Exists("app.mfa_required"))
	})

	t.Run("missing key is ignored", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("auth.providers.local.type", "local"))

		migrateGlobalMFARequired(k)

		assert.False(t, k.Exists("app.mfa_required"))
		assert.False(t, k.Exists("auth.providers.local.mfa_required"))
	})
}
