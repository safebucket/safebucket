package configuration

import (
	"fmt"
	"os"
	"strings"

	"api/internal/models"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

func parseArrayFields(k *koanf.Koanf) {
	for _, field := range ArrayConfigFields {
		if stringVal := k.String(field); stringVal != "" {
			stringVal = strings.Trim(stringVal, "[]")
			var items []string
			if strings.Contains(stringVal, ",") {
				items = strings.Split(stringVal, ",")
			} else {
				items = strings.Fields(stringVal)
			}
			for i, item := range items {
				items[i] = strings.TrimSpace(item)
			}
			err := k.Set(field, items)
			if err != nil {
				zap.L().
					Error("Error parsing array field", zap.String("field", field), zap.Error(err))
			}
		}
	}
}

func parseAuthProviders(k *koanf.Koanf) {
	providersStr := k.String("auth.providers.keys")
	if providersStr != "" {
		providers := strings.Split(providersStr, ",")
		for _, provider := range providers {
			providerUpper := strings.ToUpper(provider)
			typeKey := fmt.Sprintf("AUTH__PROVIDERS__%s__TYPE", providerUpper)
			providerType := strings.ToUpper(os.Getenv(typeKey))

			for _, key := range AuthProviderKeys {
				keyUpper := strings.ToUpper(key)
				envKey := fmt.Sprintf(
					"AUTH__PROVIDERS__%s__%s__%s",
					providerUpper,
					providerType,
					keyUpper,
				)
				if envVal := os.Getenv(envKey); envVal != "" {
					err := k.Set(
						fmt.Sprintf("auth.providers.%s.%s.%s", provider, providerType, key),
						envVal,
					)
					if err != nil {
						zap.L().
							Error("Failed to unmarshal value", zap.Error(err), zap.String("key", key))
					}
				}
			}
		}
		// Remove the keys entry to avoid conflict with providers map
		k.Delete("auth.providers.keys")
	}
}

func readEnvVars(k *koanf.Koanf) {
	err := k.Load(env.Provider("", ".", func(s string) string {
		s = strings.ToLower(s)
		segments := strings.Split(s, "__")
		result := strings.Join(segments, ".")
		return result
	}), nil)
	if err != nil {
		zap.L().Warn("Error loading environment variables", zap.Error(err))
	}

	parseArrayFields(k)
	parseAuthProviders(k)
}

func readFileConfig(k *koanf.Koanf) {
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	var filePath string
	if configFilePath == "" {
		for _, path := range ConfigFileSearchPaths {
			if _, err := os.Stat(path); err == nil {
				filePath = path
				break
			}
		}
	} else {
		filePath = configFilePath
	}

	if filePath != "" {
		err := k.Load(file.Provider(filePath), yaml.Parser())
		if err != nil {
			zap.L().
				Fatal("Fatal error loading config file", zap.String("path", filePath), zap.Error(err))
		}
		zap.L().Info("Read configuration from file " + filePath)
	} else {
		zap.L().Warn("No configuration file found")
	}
}

func loadDefaults(k *koanf.Koanf) {
	defaults := map[string]interface{}{
		"app.profile":                "default",
		"app.access_token_expiry":    60,
		"app.refresh_token_expiry":   600,
		"app.mfa_token_expiry":       5,
		"app.log_level":              "info",
		"app.port":                   8080,
		"app.trash_retention_days":   7,
		"app.max_upload_size":        int64(53687091200),
		"app.static_files.enabled":   true,
		"app.static_files.directory": "web/dist",

		"database.port": int32(5432),

		"notifier.smtp.enable_tls":      false,
		"notifier.smtp.skip_verify_tls": false,
	}

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		zap.L().Fatal("Failed to load default configuration", zap.Error(err))
	}
}

func setIfMissing(k *koanf.Koanf, key string, value interface{}) {
	if !k.Exists(key) {
		_ = k.Set(key, value)
	}
}

func loadConditionalDefaults(k *koanf.Koanf) {
	if k.String("storage.type") == "s3" {
		setIfMissing(k, "storage.s3.region", "us-east-1")
		setIfMissing(k, "storage.s3.force_path_style", true)
		setIfMissing(k, "storage.s3.use_tls", true)
	}
	if k.String("events.type") == "gcp" {
		setIfMissing(k, "events.gcp.subscription_suffix", "-sub")
	}
}

func Read() models.Configuration {
	k := koanf.New(".")

	loadDefaults(k)
	readFileConfig(k)
	readEnvVars(k)
	loadConditionalDefaults(k)

	var config models.Configuration
	err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "mapstructure"})
	if err != nil {
		zap.L().Fatal("Unable to decode config into struct", zap.Error(err))
	}

	validate := validator.New()
	if err = validate.Struct(config); err != nil {
		zap.L().Fatal("Invalid configuration", zap.Error(err))
	}

	return config
}
