package main

import (
	"api/internal/configuration"
	"api/internal/core"
	"api/internal/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	config := configuration.Read()
	core.NewLogger(config.App.LogLevel)

	profile := configuration.GetProfile(config.Profile)

	db := database.InitDB(config.Database)
	cache := core.NewCache(config.Cache)
	store := core.NewStorage(config.Storage, config.App.TrashRetentionDays)
	notify := core.NewNotifier(config.Notifier)
	activityLogger := core.NewActivityLogger(config.Activity)

	var eventsManager *core.EventsManager
	var eventRouter *core.EventRouter
	if profile.NeedsEvents() {
		eventsManager = core.NewEventsManager(config.Events, store)
		eventRouter = core.NewEventRouter(eventsManager)
	}

	if profile.HTTPServer {
		core.CreateAdminUser(db, config)
	}

	appIdentity := uuid.New().String()

	if cache != nil {
		go cache.StartIdentityTicker(appIdentity)
		zap.L().Info("Cache identity ticker started")
	}

	if profile.Workers.AnyEnabled() {
		core.StartWorkers(
			profile,
			eventsManager,
			db,
			store,
			activityLogger,
			notify,
			eventRouter,
			config,
			cache,
			appIdentity,
		)
	}

	if profile.HTTPServer {
		core.StartHTTPServer(config, db, cache, store, activityLogger, notify, eventRouter)
	} else if profile.Workers.AnyEnabled() {
		zap.L().Info("Running in worker-only mode")
		select {} // Block forever
	}
}
