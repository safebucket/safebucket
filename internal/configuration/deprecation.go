package configuration

import (
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

type deprecatedKey struct {
	oldKey string
	newKey string
}

var deprecatedKeys = []deprecatedKey{
	{oldKey: "database.host", newKey: "database.postgres.host"},
	{oldKey: "database.port", newKey: "database.postgres.port"},
	{oldKey: "database.user", newKey: "database.postgres.user"},
	{oldKey: "database.password", newKey: "database.postgres.password"},
	{oldKey: "database.name", newKey: "database.postgres.name"},
	{oldKey: "database.sslmode", newKey: "database.postgres.sslmode"},
}

// migrateDeprecatedKeys remaps deprecated config keys to their new paths.
// If only the old key is set, its value is copied to the new key with a warning.
// If both are set, the new key takes precedence and the old key is ignored.
func migrateDeprecatedKeys(k *koanf.Koanf) {
	for _, dk := range deprecatedKeys {
		oldExists := k.Exists(dk.oldKey)
		newExists := k.Exists(dk.newKey)

		switch {
		case oldExists && !newExists:
			if err := k.Set(dk.newKey, k.Get(dk.oldKey)); err != nil {
				zap.L().Error("Failed to migrate deprecated key",
					zap.String("old_key", dk.oldKey), zap.String("new_key", dk.newKey), zap.Error(err))
				continue
			}
			k.Delete(dk.oldKey)
			zap.L().Warn("Deprecated configuration key used, please migrate",
				zap.String("old_key", dk.oldKey), zap.String("new_key", dk.newKey))
		case oldExists && newExists:
			zap.L().Warn("Deprecated configuration key ignored, new key takes precedence",
				zap.String("old_key", dk.oldKey), zap.String("new_key", dk.newKey))
		}
	}
}
