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
	case DialectPostgres:
		migrationsFS = postgresMigrations
	case DialectSQLite:
		migrationsFS = sqliteMigrations
	}

	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	migrationsDir := fmt.Sprintf("migrations/%s", dialect)

	if err := goose.Up(db, migrationsDir); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.String("dialect", dialect), zap.Error(err))
	}
}
