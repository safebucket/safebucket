package core

import (
	"context"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BootOptions struct {
	DB *gorm.DB
}

type BootedApp struct {
	Config         models.Configuration
	Profile        models.Profile
	DB             *gorm.DB
	Cache          c.ICache
	Storage        storage.IStorage
	Notifier       notifier.INotifier
	ActivityLogger activity.IActivityLogger
	EventsManager  *EventsManager
	EventRouter    *EventRouter
	Workers        *WorkersHandle
	Router         chi.Router
	AppIdentity    string
}

func Boot(ctx context.Context, cfg models.Configuration, opts BootOptions) *BootedApp {
	profile := configuration.GetProfile(cfg.App.Profile)

	db := opts.DB
	if db == nil {
		db = NewDatabase(cfg.Database)
	}

	cache := NewCache(cfg.Cache)
	store := NewStorage(cfg.Storage, cfg.App.TrashRetentionDays)
	notify := NewNotifier(cfg.Notifier)
	activityLogger := NewActivityLogger(cfg.Activity)

	var eventsManager *EventsManager
	var eventRouter *EventRouter
	if profile.NeedsEvents() {
		eventsManager = NewEventsManager(cfg.Events, cfg.Storage.Type, store)
		eventRouter = NewEventRouter(eventsManager)
	}

	if profile.HTTPServer {
		CreateAdminUser(db, cfg)
	}

	workers := NewWorkersHandle()
	appIdentity := uuid.New().String()
	StartIdentityTicker(ctx, workers.WG(), cache, appIdentity)

	if profile.Workers.AnyEnabled() {
		StartWorkers(
			ctx,
			workers,
			profile,
			eventsManager,
			db,
			store,
			activityLogger,
			notify,
			eventRouter,
			cfg,
			cache,
			appIdentity,
		)
	}

	providers := configuration.LoadProviders(ctx, cfg.App.APIURL, cfg.Auth.Providers)
	router := BuildAPIRouter(cfg, db, cache, store, activityLogger, notify, eventRouter, providers)

	return &BootedApp{
		Config:         cfg,
		Profile:        profile,
		DB:             db,
		Cache:          cache,
		Storage:        store,
		Notifier:       notify,
		ActivityLogger: activityLogger,
		EventsManager:  eventsManager,
		EventRouter:    eventRouter,
		Workers:        workers,
		Router:         router,
		AppIdentity:    appIdentity,
	}
}

func (a *BootedApp) Shutdown(ctx context.Context) error {
	if a.EventsManager != nil {
		a.EventsManager.Close()
	}
	if a.Workers != nil {
		if err := a.Workers.Shutdown(ctx); err != nil {
			zap.L().Error("Workers shutdown error", zap.Error(err))
			return err
		}
	}
	return nil
}
