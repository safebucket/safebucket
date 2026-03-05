package configuration

import (
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

// migrateDeprecatedKeys remaps deprecated config keys to their new paths.
// If only the old key is set, its value is copied to the new key with a warning.
// If both are set, the new key takes precedence and the old key is ignored.
func migrateDeprecatedKeys(k *koanf.Koanf) {
	deprecatedKeys := []struct {
		oldKey string
		newKey string
	}{
		{"database.ssl_mode", "database.postgres.ssl_mode"},
		{"database.max_idle_conns", "database.postgres.max_idle_conns"},
		{"database.max_open_conns", "database.postgres.max_open_conns"},
		{"app.enable_static_files", "app.static_files.enabled"},
		{"app.static_files_dir", "app.static_files.directory"},
	}

	for _, dk := range deprecatedKeys {
		oldExists := k.Exists(dk.oldKey)
		newExists := k.Exists(dk.newKey)

		if oldExists && !newExists {
			if err := k.Set(dk.newKey, k.Get(dk.oldKey)); err != nil {
				zap.L().Error("Failed to migrate deprecated configuration key", zap.Error(err))
				continue
			}
			k.Delete(dk.oldKey)

			zap.L().
				Warn(
					"Deprecated configuration key used, please migrate",
					zap.String("old_key", dk.oldKey),
					zap.String("new_key", dk.newKey),
				)
		} else if oldExists && newExists {
			zap.L().
				Warn("Deprecated configuration key ignored, new key takes precedence",
					zap.String("old_key", dk.oldKey), zap.String("new_key", dk.newKey))
		}
	}
}
