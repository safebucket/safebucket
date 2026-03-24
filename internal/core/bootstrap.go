package core

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/events"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/services"
	"github.com/safebucket/safebucket/internal/storage"
	"github.com/safebucket/safebucket/internal/workers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateAdminUser(db *gorm.DB, config models.Configuration) {
	adminUser := models.User{
		FirstName:    "admin",
		LastName:     "admin",
		Email:        config.App.AdminEmail,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleAdmin,
	}

	hash, err := h.CreateHash(config.App.AdminPassword)
	if err != nil {
		zap.L().Fatal("Failed to hash admin password", zap.Error(err))
	}
	adminUser.HashedPassword = hash

	database.UpsertAdminUser(db, &adminUser)
}

func StartWorkers(
	profile models.Profile,
	eventsManager *EventsManager,
	db *gorm.DB,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	notify notifier.INotifier,
	eventRouter *EventRouter,
	config models.Configuration,
	cache c.ICache,
	appIdentity string,
) {
	eventParams := &events.EventParams{
		WebURL:             config.App.WebURL,
		Notifier:           notify,
		Publisher:          eventRouter,
		DB:                 db,
		Storage:            store,
		ActivityLogger:     activityLogger,
		TrashRetentionDays: config.App.TrashRetentionDays,
		Cache:              cache,
	}

	events.StartFileNotificationBuffer(cache, notify)
	zap.L().Info("Started file notification buffer")

	notifications := eventsManager.GetSubscriber(configuration.EventsNotifications).Subscribe()
	go events.HandleEvents(eventParams, notifications)
	zap.L().Info("Started notifications worker")

	if RequiresUploadConfirmation(config.Storage.Type, config.Events.Type) {
		startWorker(profile.Workers.TrashCleanup, "trash_cleanup", cache, appIdentity, func(ctx context.Context) {
			worker := &workers.TrashCleanupWorker{
				DB:                 db,
				Publisher:          eventRouter,
				TrashRetentionDays: config.App.TrashRetentionDays,
				RunInterval:        time.Duration(config.App.TrashRetentionDays) * 24 * time.Hour / 7,
			}
			worker.Start(ctx)
		})
	}

	startWorker(profile.Workers.GarbageCollector, "garbage_collector", cache, appIdentity, func(ctx context.Context) {
		worker := &workers.GarbageCollectorWorker{
			DB:             db,
			Storage:        store,
			ActivityLogger: activityLogger,
			RunInterval:    15 * time.Minute,
		}
		worker.Start(ctx)
	})

	startWorker(profile.Workers.ObjectDeletion, "object_deletion", cache, appIdentity, func(_ context.Context) {
		deletionEvents := eventsManager.GetSubscriber(configuration.EventsObjectDeletion).Subscribe()
		events.HandleEvents(eventParams, deletionEvents)
	})

	startWorker(profile.Workers.BucketEvents, "bucket_events", cache, appIdentity, func(_ context.Context) {
		bucketEventsSubscriber := eventsManager.GetSubscriber(configuration.EventsBucketEvents)
		bucketEvents := bucketEventsSubscriber.Subscribe()
		events.HandleBucketEvents(
			eventsManager.parser,
			db,
			activityLogger,
			store,
			eventRouter,
			config.App.TrashRetentionDays,
			bucketEvents,
		)
	})
}

func startWorker(
	mode models.WorkerMode,
	workerName string,
	cache c.ICache,
	appIdentity string,
	runWorker func(context.Context),
) {
	if mode == models.WorkerModeDisabled {
		return
	}

	if mode == models.WorkerModeSingleton {
		go startSingletonWorker(cache, appIdentity, workerName, runWorker)
	} else {
		go runWorker(context.Background())
		zap.L().Info("Started worker", zap.String("worker", workerName))
	}
}

func startSingletonWorker(cache c.ICache, instanceID string, workerName string, runWorker func(context.Context)) {
	lockKey := fmt.Sprintf(configuration.CacheAppWorkerLockKey, workerName)
	lockTTL := time.Duration(configuration.CacheAppWorkerLockTTL) * time.Second
	ticker := time.NewTicker(time.Duration(configuration.CacheAppWorkerLockRefresh) * time.Second)
	defer ticker.Stop()

	var workerStarted bool
	var cancelWorker context.CancelFunc

	for {
		if !workerStarted {
			acquired, err := cache.SetNX(lockKey, instanceID, lockTTL)
			if err != nil {
				zap.L().Error("Failed to acquire worker lock", zap.String("worker", workerName), zap.Error(err))
			}

			if acquired {
				zap.L().Info("Acquired worker lock, starting worker", zap.String("worker", workerName))
				workerStarted = true
				var ctx context.Context
				ctx, cancelWorker = context.WithCancel(context.Background())
				go runWorker(ctx)
			}
		} else {
			refreshed, err := c.RefreshLock(cache, lockKey, instanceID, lockTTL)
			if err != nil || !refreshed {
				zap.L().Warn("Lost worker lock, stopping worker", zap.String("worker", workerName))
				workerStarted = false
				if cancelWorker != nil {
					cancelWorker()
					cancelWorker = nil
				}
			}
		}

		<-ticker.C
	}
}

func StartIdentityTicker(cache c.ICache, id string) {
	register := func() error {
		return cache.ZAdd(
			configuration.CacheAppIdentityKey,
			float64(time.Now().Unix()),
			id,
		)
	}
	cleanup := func() error {
		cutoff := float64(time.Now().Unix()) - float64(configuration.CacheMaxAppIdentityLifetime)
		return cache.ZRemRangeByScore(
			configuration.CacheAppIdentityKey,
			"-inf",
			fmt.Sprintf("%f", cutoff),
		)
	}

	if err := register(); err != nil {
		zap.L().Fatal("Failed to register platform", zap.String("platform", id), zap.Error(err))
	}
	if err := cleanup(); err != nil {
		zap.L().Fatal("Failed to delete inactive platforms", zap.String("platform", id), zap.Error(err))
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := register(); err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
		if err := cleanup(); err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
	}
}

func StartHTTPServer(
	config models.Configuration,
	db *gorm.DB,
	cache c.ICache,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	notify notifier.INotifier,
	eventRouter *EventRouter,
	embeddedWebFS embed.FS,
) {
	m.InitValidator(config.App.MaxUploadSize)

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

	authConfig := config.App.GetAuthConfig()

	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(m.Authenticate(authConfig.JWTSecret, cache, configuration.RefreshTokenExpiry))
		apiRouter.Use(m.AudienceValidate)
		apiRouter.Use(m.MFAValidate(db, authConfig.MFARequired))
		apiRouter.Use(m.RateLimit(
			cache,
			config.App.TrustedProxies,
			config.App.AuthenticatedRequestsPerMinute,
			config.App.UnauthenticatedRequestsPerMinute,
		))

		userService := services.UserService{
			DB:                 db,
			Cache:              cache,
			AuthConfig:         authConfig,
			Publisher:          eventRouter,
			Notifier:           notify,
			ActivityLogger:     activityLogger,
			RefreshTokenExpiry: configuration.RefreshTokenExpiry,
		}

		apiRouter.Mount("/v1/users", userService.Routes())

		apiRouter.Mount("/v1/mfa", services.MFAService{
			DB:             db,
			Cache:          cache,
			AuthConfig:     authConfig,
			Publisher:      eventRouter,
			Notifier:       notify,
			ActivityLogger: activityLogger,
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
			Cache:          cache,
			AuthConfig:     authConfig,
			Providers:      providers,
			Publisher:      eventRouter,
			ActivityLogger: activityLogger,
		}.Routes())

		apiRouter.Mount("/v1/invites", services.InviteService{
			DB:             db,
			Cache:          cache,
			Storage:        store,
			AuthConfig:     authConfig,
			Publisher:      eventRouter,
			ActivityLogger: activityLogger,
			Providers:      providers,
		}.Routes())

		apiRouter.Mount("/v1/admin", services.AdminService{
			DB:             db,
			ActivityLogger: activityLogger,
		}.Routes())

		apiRouter.Mount("/v1/shares", services.PublicShareService{
			DB:             db,
			Storage:        store,
			ActivityLogger: activityLogger,
			JWTSecret:      authConfig.JWTSecret,
		}.Routes())
	})

	if config.App.StaticFiles.Enabled {
		webFS, err := fs.Sub(embeddedWebFS, "web/dist")
		if err != nil {
			zap.L().Fatal("failed to create sub-filesystem for web assets", zap.Error(err))
		}
		staticFileService, err := services.NewStaticFileService(
			webFS,
			config.App.APIURL,
			config.Storage.GetExternalURL(),
			RequiresUploadConfirmation(config.Storage.Type, config.Events.Type),
		)
		if err != nil {
			zap.L().Fatal("failed to initialize static file service", zap.Error(err))
		}
		r.Mount("/", staticFileService.Routes())
		zap.L().Info("static file service enabled", zap.String("source", "embedded"))
	} else {
		zap.L().Info("static file service disabled")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.App.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	var err error
	if config.App.TLSCertFile != "" {
		zap.L().Info("TLS certificates provided, starting HTTPS server",
			zap.Int("port", config.App.Port),
			zap.String("cert", config.App.TLSCertFile),
		)
		err = server.ListenAndServeTLS(config.App.TLSCertFile, config.App.TLSKeyFile)
	} else {
		zap.L().Info("HTTP server starting", zap.Int("port", config.App.Port))
		err = server.ListenAndServe()
	}
	if err != nil {
		zap.L().Error("Failed to start the app", zap.Error(err))
	}
}
