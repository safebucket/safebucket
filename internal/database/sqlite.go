package database

import (
	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const sqliteMaxOpenConns = 0

func InitSQLite(config *models.SQLiteDatabaseConfig) *gorm.DB {
	dsn := config.Path + "?_txlock=immediate&_fk=on&_journal=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-2000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to SQLite", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Failed to retrieve raw SQL database", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(sqliteMaxOpenConns)

	RunMigrations(sqlDB, DialectSQLite)
	RegisterCallbacks(db)

	return db
}
