package main

import (
	"embed"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/core"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

//go:embed all:web/dist
var webDistFS embed.FS

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	config := configuration.Read()
	core.NewLogger(config.App.LogLevel)

	profile, stopProfiler := core.StartProfiler(config)
	defer stopProfiler()

	stopTracer := core.StartTracer(config)
	defer stopTracer()

	db := core.NewDatabase(config.Database)
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}
	cache := core.NewCache(config.Cache)
	defer cache.Close()
	storage := core.NewStorage(config.Storage, config.App.TrashRetentionDays)
	notify := core.NewNotifier(config.Notifier)
	activityLogger := core.NewActivityLogger(config.Activity)
	defer activityLogger.Close()

	var eventsManager *core.EventsManager
	var eventRouter *core.EventRouter
	if profile.NeedsEvents() {
		eventsManager = core.NewEventsManager(config.Events, config.Storage.Type, storage)
		eventRouter = core.NewEventRouter(eventsManager)
	}

	if profile.HTTPServer {
		core.CreateAdminUser(db, config)
	}

	appIdentity := uuid.New().String()
	go core.StartIdentityTicker(cache, appIdentity)

	if profile.Workers.AnyEnabled() {
		core.StartWorkers(
			profile,
			eventsManager,
			db,
			storage,
			activityLogger,
			notify,
			eventRouter,
			config,
			cache,
			appIdentity,
		)
	}

	if profile.HTTPServer {
		core.StartHTTPServer(config, db, cache, storage, activityLogger, notify, eventRouter, webDistFS)
	} else if profile.Workers.AnyEnabled() {
		zap.L().Info("Running in worker-only mode")
		select {} // Block forever
	}
}
