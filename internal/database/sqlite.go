package database

import (
	"context"

	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

	if _, err = sqlDB.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); err != nil {
		zap.L().Fatal("Failed to enable foreign keys", zap.Error(err))
	}

	// WAL mode enables concurrent reads. If it fails (e.g., NFS-mounted filesystem),
	// fall back to the default journal mode rather than crashing.
	if _, err = sqlDB.ExecContext(context.Background(), "PRAGMA journal_mode = WAL"); err != nil {
		zap.L().Warn("Failed to set WAL journal mode, continuing with default journal mode", zap.Error(err))
	}

	// Single connection avoids "database is locked" errors. This intentionally
	// serializes all DB operations (reads and writes) for the lite deployment target.
	// WAL mode concurrent reads are not leveraged, but correctness is prioritized
	// over throughput for single-node/self-hosted setups.
	sqlDB.SetMaxOpenConns(1)

	runMigrations(sqlDB, DialectSQLite)
	RegisterCallbacks(db)

	return db
}
