package database

import (
	"api/internal/models"
	"fmt"

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
		zap.L().Error("Failed to connect to database", zap.Error(err))
	}

	runMigrations(db)

	return db
}

func runMigrations(db *gorm.DB) {
	err := db.AutoMigrate(&models.User{}, &models.Bucket{}, &models.File{}, &models.Invite{})
	if err != nil {
		zap.L().Error("failed to migrate db models", zap.Error(err))
	}
}
