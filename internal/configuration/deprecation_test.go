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
		require.NoError(t, k.Set("database.host", "localhost"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "localhost", k.String("database.postgres.host"))
		assert.False(t, k.Exists("database.host"))
	})

	t.Run("both keys set keeps new key value", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.host", "old-host"))
		require.NoError(t, k.Set("database.postgres.host", "new-host"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "new-host", k.String("database.postgres.host"))
	})

	t.Run("only new key set remains unchanged", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.postgres.host", "new-host"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "new-host", k.String("database.postgres.host"))
		assert.False(t, k.Exists("database.host"))
	})

	t.Run("neither key set remains unchanged", func(t *testing.T) {
		k := koanf.New(".")

		migrateDeprecatedKeys(k)

		assert.False(t, k.Exists("database.host"))
		assert.False(t, k.Exists("database.postgres.host"))
	})

	t.Run("migrates all deprecated keys", func(t *testing.T) {
		k := koanf.New(".")
		require.NoError(t, k.Set("database.host", "localhost"))
		require.NoError(t, k.Set("database.port", 5432))
		require.NoError(t, k.Set("database.user", "admin"))
		require.NoError(t, k.Set("database.password", "secret"))
		require.NoError(t, k.Set("database.name", "mydb"))
		require.NoError(t, k.Set("database.sslmode", "disable"))

		migrateDeprecatedKeys(k)

		assert.Equal(t, "localhost", k.String("database.postgres.host"))
		assert.Equal(t, int64(5432), k.Int64("database.postgres.port"))
		assert.Equal(t, "admin", k.String("database.postgres.user"))
		assert.Equal(t, "secret", k.String("database.postgres.password"))
		assert.Equal(t, "mydb", k.String("database.postgres.name"))
		assert.Equal(t, "disable", k.String("database.postgres.sslmode"))
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
		require.NoError(t, k.Set("database.host", "localhost"))

		migrateDeprecatedKeys(k)

		entries := logs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, zap.WarnLevel, entries[0].Level)
		assert.Equal(t, "Deprecated configuration key used, please migrate", entries[0].Message)
		assert.Equal(t, "database.host", entries[0].ContextMap()["old_key"])
	})

	t.Run("logs warning when both keys are present", func(t *testing.T) {
		logs.TakeAll()
		k := koanf.New(".")
		require.NoError(t, k.Set("database.host", "old-host"))
		require.NoError(t, k.Set("database.postgres.host", "new-host"))

		migrateDeprecatedKeys(k)

		entries := logs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, zap.WarnLevel, entries[0].Level)
		assert.Equal(t, "Deprecated configuration key ignored, new key takes precedence", entries[0].Message)
	})
}
