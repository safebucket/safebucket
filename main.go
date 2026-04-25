package main

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/core"
	"github.com/safebucket/safebucket/internal/middlewares"

	"go.uber.org/zap"
)

//go:embed all:web/dist
var webDistFS embed.FS

const httpShutdownTimeout = 10 * time.Second

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	config := configuration.Read()
	core.NewLogger(config.App.LogLevel)

	defer core.StartProfiler(config)()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	middlewares.InitValidator(config.App.MaxUploadSize)

	stopTracer := core.StartTracer(config)
	defer stopTracer()

	app := core.Boot(ctx, config, core.BootOptions{})
	if sqlDB, err := app.DB.DB(); err == nil {
		defer sqlDB.Close()
	}
	defer app.Cache.Close()
	defer app.ActivityLogger.Close()

	var httpShutdown func(context.Context) error
	var httpErr <-chan error
	if app.Profile.HTTPServer {
		httpShutdown, httpErr = core.StartHTTPServer(config, app.Router, webDistFS)
	} else if app.Profile.Workers.AnyEnabled() {
		zap.L().Info("Running in worker-only mode")
	}

	select {
	case <-ctx.Done():
		zap.L().Info("Shutdown signal received")
	case err := <-httpErr:
		if err != nil {
			zap.L().Error("HTTP server failed to start, shutting down", zap.Error(err))
		}
		cancel()
	}

	if httpShutdown != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		if err := httpShutdown(shutdownCtx); err != nil {
			zap.L().Error("HTTP server shutdown error", zap.Error(err))
		}
		shutdownCancel()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	if err := app.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("App shutdown error", zap.Error(err))
	}
	shutdownCancel()
}
