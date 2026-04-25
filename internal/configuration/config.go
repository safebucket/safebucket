package configuration

import (
	"fmt"
	"os"
	"strings"

	"github.com/safebucket/safebucket/internal/models"

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

func discoverConfigFile() string {
	if p := os.Getenv("CONFIG_FILE_PATH"); p != "" {
		return p
	}
	for _, path := range ConfigFileSearchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func loadFile(k *koanf.Koanf, path string) error {
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return fmt.Errorf("load config file %s: %w", path, err)
	}
	zap.L().Info("Read configuration from file " + path)
	return nil
}

func loadDefaults(k *koanf.Koanf) {
	defaults := map[string]interface{}{
		"app.profile":                             "default",
		"app.access_token_expiry":                 60,
		"app.refresh_token_expiry":                600,
		"app.mfa_token_expiry":                    5,
		"app.log_level":                           "info",
		"app.port":                                8080,
		"app.trash_retention_days":                7,
		"app.max_upload_size":                     int64(53687091200),
		"app.authenticated_requests_per_minute":   200,
		"app.unauthenticated_requests_per_minute": 20,
		"app.static_files.enabled":                true,
		"tracing.enabled":                         false,
		"profiling.enabled":                       false,
		"database.type":                           ProviderPostgres,
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
	if k.String("database.type") == ProviderPostgres {
		setIfMissing(k, "database.postgres.port", int32(5432))
	}
	if k.String("storage.type") == "minio" {
		setIfMissing(k, "storage.minio.region", "us-east-1")
	}
	if k.String("storage.type") == "rustfs" {
		setIfMissing(k, "storage.rustfs.region", "us-east-1")
	}
	if k.String("storage.type") == "s3" {
		setIfMissing(k, "storage.s3.region", "us-east-1")
		setIfMissing(k, "storage.s3.force_path_style", true)
		setIfMissing(k, "storage.s3.use_tls", true)
	}
	if k.String("events.type") == "gcp" {
		setIfMissing(k, "events.gcp.subscription_suffix", "-sub")
	}
	if k.String("notifier.type") == "smtp" {
		setIfMissing(k, "notifier.smtp.tls_mode", models.TLSModeStartTLS)
		setIfMissing(k, "notifier.smtp.skip_verify_tls", false)
	}
	if k.String("profiling.type") == "pyroscope" {
		setIfMissing(k, "profiling.pyroscope.application_name", AppName)
		setIfMissing(k, "profiling.pyroscope.upload_rate", 15)
	}
	if k.String("tracing.type") == "tempo" {
		setIfMissing(k, "tracing.tempo.service_name", AppName)
		setIfMissing(k, "tracing.tempo.sampling_rate", 1.0)
	}
}

type LoadOptions struct {
	ConfigFilePath string
	SkipEnv        bool
}

func Load(opts LoadOptions) (models.Configuration, error) {
	k := koanf.New(".")
	loadDefaults(k)

	path := opts.ConfigFilePath
	if path == "" {
		path = discoverConfigFile()
	}
	if path != "" {
		if err := loadFile(k, path); err != nil {
			return models.Configuration{}, err
		}
	} else if opts.ConfigFilePath == "" {
		zap.L().Warn("No configuration file found")
	}

	if !opts.SkipEnv {
		readEnvVars(k)
	}
	migrateDeprecatedKeys(k)
	loadConditionalDefaults(k)

	var cfg models.Configuration
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return models.Configuration{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

func Validate(cfg models.Configuration) error {
	return validator.New().Struct(cfg)
}

func Read() models.Configuration {
	cfg, err := Load(LoadOptions{})
	if err != nil {
		zap.L().Fatal("Unable to load configuration", zap.Error(err))
	}
	if validateErr := Validate(cfg); validateErr != nil {
		zap.L().Fatal("Invalid configuration", zap.Error(validateErr))
	}
	return cfg
}
