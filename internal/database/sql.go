package database

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

// DialectSQLite is the GORM dialector name for SQLite databases.
const DialectSQLite = "sqlite"

func runMigrations(db *sql.DB, dialect string) {
	gooseDialect := dialect
	if dialect == DialectSQLite {
		gooseDialect = "sqlite3"
	}

	if err := goose.SetDialect(gooseDialect); err != nil {
		zap.L().Fatal("Failed to set goose dialect", zap.String("dialect", gooseDialect), zap.Error(err))
	}

	var migrationsFS embed.FS
	switch dialect {
	case "postgres":
		migrationsFS = postgresMigrations
	case DialectSQLite:
		migrationsFS = sqliteMigrations
	}

	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	migrationsDir := "migrations/" + dialect

	if err := goose.Up(db, migrationsDir); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.String("dialect", dialect), zap.Error(err))
	}
}
