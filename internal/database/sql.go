package database

import (
	"fmt"

	"api/internal/models"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB connects to the database without running migrations.
func InitDB(config models.DatabaseConfiguration) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host, config.User, config.Password, config.Name, config.Port, config.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	return db
}

// RunMigrations runs database migrations using Goose.
func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw SQL database: %w", err)
	}
	if err = goose.Up(sqlDB, "internal/database/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	zap.L().Info("Database migrations completed successfully")
	return nil
}
