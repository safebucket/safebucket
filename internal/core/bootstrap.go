package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"api/internal/activity"
	c "api/internal/cache"
	"api/internal/configuration"
	"api/internal/events"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/notifier"
	"api/internal/services"
	"api/internal/storage"
	"api/internal/workers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	}

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
				ActivityLogger:     activityLogger,
			}
			worker.Start(ctx)
		})
	}

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
	ticker := time.NewTicker(time.Duration(configuration.CacheAppWorkerLockRefresh) * time.Second)
	defer ticker.Stop()

	var workerStarted bool
	var cancelWorker context.CancelFunc

	for {
		if !workerStarted {
			acquired, err := cache.TryAcquireLock(lockKey, instanceID, configuration.CacheAppWorkerLockTTL)
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
			refreshed, err := cache.RefreshLock(lockKey, instanceID, configuration.CacheAppWorkerLockTTL)
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

func StartHTTPServer(
	config models.Configuration,
	db *gorm.DB,
	cache c.ICache,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	notify notifier.INotifier,
	eventRouter *EventRouter,
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
		apiRouter.Use(m.Authenticate(authConfig.JWTSecret))
		apiRouter.Use(m.AudienceValidate)
		apiRouter.Use(m.MFAValidate(db, authConfig.MFARequired))
		apiRouter.Use(m.RateLimit(cache, config.App.TrustedProxies))

		userService := services.UserService{
			DB:         db,
			Cache:      cache,
			AuthConfig: authConfig,
			Publisher:  eventRouter,
			Notifier:   notify,
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
	})

	if config.App.StaticFiles.Enabled {
		staticFileService, err := services.NewStaticFileService(
			config.App.StaticFiles.Directory,
			config.App.APIURL,
			config.Storage.GetExternalURL(),
			RequiresUploadConfirmation(config.Storage.Type, config.Events.Type),
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
