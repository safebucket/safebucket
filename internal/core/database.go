package core

import (
	"api/internal/configuration"
	"api/internal/database"
	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewDatabase(config models.DatabaseConfiguration) *gorm.DB {
	switch config.Type {
	case configuration.ProviderPostgres:
		return database.InitPostgres(config.Postgres)
	case configuration.ProviderSQLite:
		return database.InitSQLite(config.SQLite)
	default:
		zap.L().Fatal("Unsupported database type", zap.String("type", config.Type))
		return nil
	}
}
