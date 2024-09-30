package configuration

import (
	"api/internal/models"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
)

func readEnvVars(v *viper.Viper) {
	zap.L().Info("Load environment variables")
	v.AutomaticEnv()
	v.BindEnv("database.host", "DATABASE_HOST")
	v.BindEnv("database.port", "DATABASE_PORT")
	v.BindEnv("database.name", "DATABASE_NAME")
	v.BindEnv("database.user", "DATABASE_USER")
	v.BindEnv("database.password", "DATABASE_PASSWORD")
	v.BindEnv("database.sslmode", "DATABASE_SSLMODE")

	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")

	v.BindEnv("auth_providers.google_client_id", "GOOGLE_CLIENT_ID")
	v.BindEnv("auth_providers.google_client_secret", "GOOGLE_CLIENT_SECRET")
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
	readEnvVars(v)
	readFileConfig(v)
	setDefault(v)

	var config models.Configuration
	err := v.Unmarshal(&config)

	if err != nil {
		zap.L().Error("Unable to decode into struct: ", zap.Error(err))
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		zap.L().Error("Invalid configuration: ", zap.Error(err))
	}

	return config
}
