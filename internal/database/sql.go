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
	var exists bool
	err := db.Raw("select exists(select 1 from pg_type where typname = 'file_status')").Scan(&exists).Error
	if err != nil {
		zap.L().Fatal("failed to check if file_status enum exists", zap.Error(err))
	}

	if !exists {
		err = db.Exec("CREATE TYPE file_status AS ENUM ('uploading', 'uploaded', 'deleting')").Error
		if err != nil {
			zap.L().Fatal("failed to create file_status enum", zap.Error(err))
		}
	}

	err = db.Raw("select exists(select 1 from pg_type where typname = 'provider_type')").Scan(&exists).Error
	if err != nil {
		zap.L().Fatal("failed to check if provider_type enum exists", zap.Error(err))
	}

	if !exists {
		err = db.Exec("CREATE TYPE provider_type AS ENUM ('local', 'oidc')").Error
		if err != nil {
			zap.L().Fatal("failed to create provider_type enum", zap.Error(err))
		}
	}

	err = db.AutoMigrate(&models.User{}, &models.Bucket{}, &models.File{}, &models.Invite{}, &models.Challenge{})
	if err != nil {
		zap.L().Fatal("failed to migrate db models", zap.Error(err))
	}
}
