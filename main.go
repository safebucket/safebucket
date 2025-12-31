package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"api/internal/activity"
	c "api/internal/cache"
	"api/internal/configuration"
	"api/internal/core"
	"api/internal/database"
	"api/internal/events"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/notifier"
	"api/internal/services"
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	config, profile := configuration.Read()
	core.NewLogger(config.App.LogLevel)

	logProfile(profile)

	// Database is always initialized
	db := database.InitDB(config.Database)

	// Migrations (conditional based on profile)
	if profile.Migrations {
		if err := database.RunMigrations(db); err != nil {
			zap.L().Fatal("migrations failed", zap.Error(err))
		}
	}

	// Exit after init for migrate profile
	if profile.ExitAfterInit {
		zap.L().Info("Profile complete, exiting", zap.String("profile", profile.Name))
		return
	}

	// Initialize dependencies based on profile requirements
	var cache c.ICache
	if profile.NeedsCache() {
		cache = core.NewCache(config.Cache)
	}

	var store storage.IStorage
	if profile.NeedsStorage() {
		store = core.NewStorage(config.Storage, config.App.TrashRetentionDays)
	}

	var notify notifier.INotifier
	if profile.NeedsNotifier() {
		notify = core.NewNotifier(config.Notifier)
	}

	var activityLogger activity.IActivityLogger
	if profile.NeedsActivity() {
		activityLogger = core.NewActivityLogger(config.Activity)
	}

	// Events infrastructure (needed for workers or HTTP server publishing)
	var eventsManager *core.EventsManager
	var eventRouter *core.EventRouter
	if profile.NeedsEvents() {
		eventsManager = core.NewEventsManager(config.Events, store)
		eventRouter = core.NewEventRouter(eventsManager)
	}

	// Admin user creation (only for HTTP profiles)
	if profile.HTTPServer {
		createAdminUser(db, config)
	}

	appIdentity := uuid.New().String()

	// Start workers based on profile
	if profile.Workers.AnyEnabled() {
		startWorkers(profile, eventsManager, db, store, activityLogger, notify, eventRouter, config)
	}

	// Cache ticker
	if profile.CacheTicker && cache != nil {
		go cache.StartIdentityTicker(appIdentity)
		zap.L().Info("Cache identity ticker started")
	}

	// HTTP server
	if profile.HTTPServer {
		startHTTPServer(config, db, cache, store, activityLogger, eventRouter)
	} else if profile.Workers.AnyEnabled() {
		zap.L().Info("Running in worker-only mode")
		select {} // Block forever
	}
}

func logProfile(profile models.Profile) {
	zap.L().Info("=== SafeBucket Profile ===",
		zap.String("name", profile.Name),
		zap.Bool("http_server", profile.HTTPServer),
		zap.Bool("migrations", profile.Migrations),
		zap.Bool("cache_ticker", profile.CacheTicker),
		zap.Bool("worker:notifications", profile.Workers.Notifications),
		zap.Bool("worker:object_deletion", profile.Workers.ObjectDeletion),
		zap.Bool("worker:bucket_events", profile.Workers.BucketEvents),
	)
}

func createAdminUser(db *gorm.DB, config models.Configuration) {
	adminUser := models.User{
		FirstName:    "admin",
		LastName:     "admin",
		Email:        config.App.AdminEmail,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleAdmin,
	}

	hash, _ := h.CreateHash(config.App.AdminPassword)
	adminUser.HashedPassword = hash
	db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "email"}, {Name: "provider_key"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: "deleted_at", Value: nil},
		}},
		DoUpdates: clause.AssignmentColumns([]string{"hashed_password"}),
	}).Create(&adminUser)
}

func startWorkers(
	profile models.Profile,
	eventsManager *core.EventsManager,
	db *gorm.DB,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	notify notifier.INotifier,
	eventRouter *core.EventRouter,
	config models.Configuration,
) {
	eventParams := &events.EventParams{
		WebURL:             config.App.WebURL,
		Notifier:           notify,
		Publisher:          eventRouter,
		DB:                 db,
		Storage:            store,
		ActivityLogger:     activityLogger,
		TrashRetentionDays: config.App.TrashRetentionDays,
	}

	if profile.Workers.Notifications {
		notifications := eventsManager.GetSubscriber(configuration.EventsNotifications).Subscribe()
		go events.HandleEvents(eventParams, notifications)
		zap.L().Info("Started notifications worker")
	}

	if profile.Workers.ObjectDeletion {
		deletionEvents := eventsManager.GetSubscriber(configuration.EventsObjectDeletion).Subscribe()
		go events.HandleEvents(eventParams, deletionEvents)
		zap.L().Info("Started object deletion worker")
	}

	if profile.Workers.BucketEvents {
		bucketEventsSubscriber := eventsManager.GetSubscriber(configuration.EventsBucketEvents)
		bucketEvents := bucketEventsSubscriber.Subscribe()
		go events.HandleBucketEvents(
			bucketEventsSubscriber,
			db,
			activityLogger,
			store,
			config.App.TrashRetentionDays,
			bucketEvents,
		)
		zap.L().Info("Started bucket events worker")
	}
}

func startHTTPServer(
	config models.Configuration,
	db *gorm.DB,
	cache c.ICache,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	eventRouter *core.EventRouter,
) {
	r := chi.NewRouter()

	r.Use(middleware.Timeout(5 * time.Second))
	r.Use(m.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.App.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	providers := configuration.LoadProviders(
		context.Background(),
		config.App.APIURL,
		config.Auth.Providers,
	)

	// API routes with auth middleware
	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(m.Authenticate(config.App.JWTSecret))
		apiRouter.Use(m.RateLimit(cache, config.App.TrustedProxies))

		apiRouter.Mount("/v1/users", services.UserService{
			DB: db,
		}.Routes())

		apiRouter.Mount("/v1/buckets", services.BucketService{
			DB:                 db,
			Storage:            store,
			Publisher:          eventRouter,
			ActivityLogger:     activityLogger,
			Providers:          providers,
			WebURL:             config.App.WebURL,
			TrashRetentionDays: config.App.TrashRetentionDays,
		}.Routes())

		apiRouter.Mount("/v1/auth", services.AuthService{
			DB:             db,
			JWTSecret:      config.App.JWTSecret,
			Providers:      providers,
			WebURL:         config.App.WebURL,
			Publisher:      eventRouter,
			ActivityLogger: activityLogger,
		}.Routes())

		apiRouter.Mount("/v1/invites", services.InviteService{
			DB:             db,
			JWTSecret:      config.App.JWTSecret,
			Storage:        store,
			Publisher:      eventRouter,
			ActivityLogger: activityLogger,
			Providers:      providers,
			WebURL:         config.App.WebURL,
		}.Routes())

		apiRouter.Mount("/v1/admin", services.AdminService{
			DB:             db,
			ActivityLogger: activityLogger,
		}.Routes())
	})

	// Initialize and mount static file service (if enabled)
	if config.App.StaticFiles.Enabled {
		staticFileService, err := services.NewStaticFileService(
			config.App.StaticFiles.Directory,
			config.App.APIURL,
			config.Storage.GetExternalURL(),
		)
		if err != nil {
			zap.L().Fatal("failed to initialize static file service", zap.Error(err))
		}
		r.Mount("/", staticFileService.Routes())
		zap.L().Info("static file service enabled", zap.String("directory", config.App.StaticFiles.Directory))
	} else {
		zap.L().Info("static file service disabled")
	}

	zap.L().Info("HTTP server starting", zap.Int("port", config.App.Port))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.App.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		zap.L().Error("Failed to start the app", zap.Error(err))
	}
}
