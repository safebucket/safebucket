package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

const DialectSQLite = "sqlite"
const DialectPostgres = "postgres"

var migrationSources = map[string]embed.FS{
	DialectPostgres: postgresMigrations,
	DialectSQLite:   sqliteMigrations,
}

func runMigrations(db *sql.DB, dialect string) {
	gooseDialect := dialect
	if dialect == DialectSQLite {
		gooseDialect = "sqlite3"
	}

	if err := goose.SetDialect(gooseDialect); err != nil {
		zap.L().Fatal("Failed to set goose dialect", zap.String("dialect", gooseDialect), zap.Error(err))
	}

	goose.SetBaseFS(migrationSources[dialect])
	defer goose.SetBaseFS(nil)

	if err := goose.Up(db, fmt.Sprintf("migrations/%s", dialect)); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.String("dialect", dialect), zap.Error(err))
	}
}
