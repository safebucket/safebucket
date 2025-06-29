package database

import (
	"api/internal/models"
	"database/sql"
	"fmt"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(config models.DatabaseConfiguration) *gorm.DB {
	dbConnectionString := getDatabaseConnectionString(config)
	db, err := gorm.Open(postgres.Open(dbConnectionString), &gorm.Config{})
	if err != nil {
		zap.L().Error("Failed to connect to database", zap.Error(err))
	}

	sqlDb, _ := db.DB()
	runMigrations(sqlDb)

	return db
}

func runMigrations(db *sql.DB) {
	if err := goose.Up(db, "internal/database/migrations"); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}
}

func getDatabaseConnectionString(config models.DatabaseConfiguration) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host, config.User, config.Password, config.Name, config.Port, config.SSLMode,
	)
}
