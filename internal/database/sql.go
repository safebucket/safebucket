package database

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

type zapGooseLogger struct{ l *zap.Logger }

func (g zapGooseLogger) Printf(format string, v ...any) {
	g.l.Sugar().Infof(strings.TrimRight(format, "\n"), v...)
}

func (g zapGooseLogger) Fatalf(format string, v ...any) {
	g.l.Sugar().Fatalf(strings.TrimRight(format, "\n"), v...)
}

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

const (
	DialectPostgres = "postgres"
	DialectSQLite   = "sqlite"
)

var migrationSources = map[string]embed.FS{
	DialectPostgres: postgresMigrations,
	DialectSQLite:   sqliteMigrations,
}

func RunMigrations(db *sql.DB, dialect string) {
	goose.SetLogger(zapGooseLogger{l: zap.L()})
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
