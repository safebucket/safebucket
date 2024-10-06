package configuration

import (
	"api/internal/models"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"strings"
)

func readEnvVars(v *viper.Viper) {
	zap.L().Info("Load environment variables")
	v.AutomaticEnv()
	_ = v.BindEnv("database.host", "DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT")
	_ = v.BindEnv("database.name", "DATABASE_NAME")
	_ = v.BindEnv("database.user", "DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD")
	_ = v.BindEnv("database.sslmode", "DATABASE_SSLMODE")

	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")

	//	keys := []string{"name", "client_id", "client_secret", "issuer"}

	_ = v.BindEnv("auth.providers.authelia.client_id", "AUTH_PROVIDER_AUTHELIA_CLIENT_ID")
	_ = v.BindEnv("auth.providers.authelia.client_secret", "AUTH_PROVIDER_AUTHELIA_CLIENT_SECRET")
	_ = v.BindEnv("auth.providers.authelia.issuer", "AUTH_PROVIDER_AUTHELIA_ISSUER")

	//zap.L().Warn(fmt.Sprintf(%v, providers))
	//	for _, provider := range providers {
	//		providerUpper := strings.ToUpper(provider)
	//		for _, key := range keys {
	//			keyUpper := strings.ToUpper(key)
	//			_ = v.BindEnv(
	//				fmt.Sprintf("auth.providers.%s.%s", provider, key),
	//				fmt.Sprintf("AUTH_PROVIDERS_%s_%s", providerUpper, keyUpper),
	//			)
	//		}
	//	}
}

func readFileConfig(v *viper.Viper) {
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("templates/")
	} else {
		v.SetConfigFile(configFilePath)
	}
	err := v.ReadInConfig()
	if err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			panic(fmt.Errorf("fatal error config file: %w", err))
		} else {
			zap.L().Warn("No configuration file found")
		}
	}
	zap.L().Info("Read configuration from file " + v.ConfigFileUsed())
}

func setDefault(v *viper.Viper) {
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.port", 5432)
}

func Read() models.Configuration {
	v := viper.New()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))

	readEnvVars(v)
	readFileConfig(v)
	setDefault(v)

	var config models.Configuration
	err := v.Unmarshal(&config)

	zap.L().Info("config", zap.Any("config auth", config.Auth.Providers)) //todo: delete

	if err != nil {
		zap.L().Error("Unable to decode into struct: ", zap.Error(err))
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		zap.L().Error("Invalid configuration: ", zap.Error(err))
	}

	return config
}
