package database

import (
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPostgres(config *models.PostgresDatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host, config.User, config.Password, config.Name, config.Port, config.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Failed to retrieve raw SQL database", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(configuration.PostgresMaxOpenConns)
	sqlDB.SetMaxIdleConns(configuration.PostgresMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(configuration.PostgresConnMaxLifetime) * time.Minute)

	RunMigrations(sqlDB, DialectPostgres)

	return db
}
