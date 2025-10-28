package database

import (
	"fmt"

	"api/internal/models"

	"database/sql"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(config models.DatabaseConfiguration) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host, config.User, config.Password, config.Name, config.Port, config.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to database for migrations", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Failed to retrieve raw SQL database", zap.Error(err))
	}

	runMigrations(sqlDB)

	return db
}

func runMigrations(db *sql.DB) {
	if err := goose.Up(db, "internal/database/migrations"); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}
}
