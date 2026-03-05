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
	t.Run("old key set and new key missing copies value", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.ssl_mode", "disable"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "disable", k.String("database.postgres.ssl_mode"))
		assert.False(t, k.Exists("database.ssl_mode"))
	})

	t.Run("both keys set keeps new key value", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.ssl_mode", "disable"))
		require.NoError(t, k.Set("database.postgres.ssl_mode", "require"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "require", k.String("database.postgres.ssl_mode"))
	})

	t.Run("only new key set remains unchanged", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.postgres.ssl_mode", "require"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "require", k.String("database.postgres.ssl_mode"))
		assert.False(t, k.Exists("database.ssl_mode"))
	})

	t.Run("neither key set remains unchanged", func(t *testing.T) {
		k := koanf.New(".")

		migrateDeprecatedKeys(k)

		assert.False(t, k.Exists("database.ssl_mode"))
		assert.False(t, k.Exists("database.postgres.ssl_mode"))
	})

	t.Run("migrates all deprecated keys", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.ssl_mode", "disable"))
		require.NoError(t, k.Set("database.max_idle_conns", 10))
		require.NoError(t, k.Set("database.max_open_conns", 100))
		require.NoError(t, k.Set("app.enable_static_files", true))
		require.NoError(t, k.Set("app.static_files_dir", "/var/www"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "disable", k.String("database.postgres.ssl_mode"))
		assert.Equal(t, int64(10), k.Int64("database.postgres.max_idle_conns"))
		assert.Equal(t, int64(100), k.Int64("database.postgres.max_open_conns"))
		assert.True(t, k.Bool("app.static_files.enabled"))
		assert.Equal(t, "/var/www", k.String("app.static_files.directory"))
	})
}

func TestMigrateDeprecatedKeys_Logging(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	restoreLogger := zap.ReplaceGlobals(logger)
	defer restoreLogger()

	t.Run("logs warning when old key is migrated", func(t *testing.T) {
		logs.TakeAll()
		k := koanf.New(".")
		require.NoError(t, k.Set("database.ssl_mode", "disable"))

		migrateDeprecatedKeys(k)

		entries := logs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, zap.WarnLevel, entries[0].Level)
		assert.Equal(t, "Deprecated configuration key used, please migrate", entries[0].Message)
		assert.Equal(t, "database.ssl_mode", entries[0].ContextMap()["old_key"])
	})

	t.Run("logs warning when both keys are present", func(t *testing.T) {
		logs.TakeAll()
		k := koanf.New(".")
		require.NoError(t, k.Set("database.ssl_mode", "disable"))
		require.NoError(t, k.Set("database.postgres.ssl_mode", "require"))

		migrateDeprecatedKeys(k)

		entries := logs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, zap.WarnLevel, entries[0].Level)
		assert.Equal(t, "Deprecated configuration key ignored, new key takes precedence", entries[0].Message)
	})
}
