package configuration

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

var renamedKeys = []struct {
	oldKey string
	newKey string
}{
	{"database.ssl_mode", "database.postgres.sslmode"},
	{"app.enable_static_files", "app.static_files.enabled"},
	{"app.jwt_secret", "app.token_secret"},
}

var removedKeys = []string{
	"database.max_idle_conns",
	"database.max_open_conns",
	"app.static_files_dir",
}

func migrateDeprecatedKeys(k *koanf.Koanf) {
	for _, rk := range renamedKeys {
		if !k.Exists(rk.oldKey) {
			continue
		}
		if k.Exists(rk.newKey) {
			zap.L().Warn("Deprecated configuration key ignored, using new key",
				zap.String("old_key", rk.oldKey), zap.String("new_key", rk.newKey))
		} else if err := k.Set(rk.newKey, k.Get(rk.oldKey)); err != nil {
			zap.L().Error("Failed to migrate deprecated configuration key", zap.Error(err))
			continue
		} else {
			zap.L().Warn("Deprecated configuration key used, please migrate",
				zap.String("old_key", rk.oldKey), zap.String("new_key", rk.newKey))
		}
		k.Delete(rk.oldKey)
	}

	for _, key := range removedKeys {
		if k.Exists(key) {
			zap.L().Warn("Configuration key is no longer supported and is ignored", zap.String("old_key", key))
			k.Delete(key)
		}
	}
}

func migrateGlobalMFARequired(k *koanf.Koanf) {
	const oldKey = "app.mfa_required"
	if !k.Exists(oldKey) {
		return
	}

	if k.Bool(oldKey) {
		for _, name := range k.MapKeys("auth.providers") {
			if k.String(fmt.Sprintf("auth.providers.%s.type", name)) != string(models.LocalProviderType) {
				continue
			}
			newKey := fmt.Sprintf("auth.providers.%s.mfa_required", name)
			if k.Exists(newKey) {
				continue
			}
			if err := k.Set(newKey, true); err != nil {
				zap.L().Error("Failed to migrate deprecated configuration key", zap.Error(err))
				continue
			}
		}

		zap.L().Warn(
			"Deprecated configuration key used, please migrate",
			zap.String("old_key", oldKey),
			zap.String("new_key", "auth.providers.<local>.mfa_required"),
		)
	}

	k.Delete(oldKey)
}
