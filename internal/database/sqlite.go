package database

import (
	"context"

	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	pragmaForeignKeys = "PRAGMA foreign_keys = ON"
	pragmaJournalWAL  = "PRAGMA journal_mode = WAL"
	pragmaBusyTimeout = "PRAGMA busy_timeout = 5000"
	pragmaCacheSize   = "PRAGMA cache_size = -2000"

	sqliteMaxOpenConns = 4
)

func InitSQLite(config *models.SQLiteDatabaseConfig) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to SQLite", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Failed to retrieve raw SQL database", zap.Error(err))
	}

	if _, err = sqlDB.ExecContext(context.Background(), pragmaForeignKeys); err != nil {
		zap.L().Fatal("Failed to enable foreign keys", zap.Error(err))
	}

	if _, err = sqlDB.ExecContext(context.Background(), pragmaJournalWAL); err != nil {
		zap.L().Fatal("Failed to set WAL journal mode", zap.Error(err))
	}

	if _, err = sqlDB.ExecContext(context.Background(), pragmaBusyTimeout); err != nil {
		zap.L().Fatal("Failed to set busy timeout", zap.Error(err))
	}

	if _, err = sqlDB.ExecContext(context.Background(), pragmaCacheSize); err != nil {
		zap.L().Fatal("Failed to set cache size", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(sqliteMaxOpenConns)

	runMigrations(sqlDB, DialectSQLite)
	RegisterCallbacks(db)

	return db
}
