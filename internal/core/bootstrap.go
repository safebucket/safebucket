package core

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/events"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/services"
	"github.com/safebucket/safebucket/internal/storage"
	"github.com/safebucket/safebucket/internal/workers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func StartProfiler(config models.Configuration) func() {
	profiler := NewProfiler(config.Profiling, config.App.Profile)
	if profiler == nil {
		return func() {}
	}
	return func() {
		if err := profiler.Stop(); err != nil {
			zap.L().Error("Failed to stop profiler", zap.Error(err))
		}
	}
}

func StartTracer(config models.Configuration) func() {
	tracer := NewTracer(config.Tracing)
	if tracer == nil {
		return func() {}
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracer.Shutdown(ctx); err != nil {
			zap.L().Error("Failed to stop tracer", zap.Error(err))
		}
	}
}

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

type WorkersHandle struct {
	wg *sync.WaitGroup
}

func NewWorkersHandle() *WorkersHandle {
	return &WorkersHandle{wg: &sync.WaitGroup{}}
}

func (h *WorkersHandle) WG() *sync.WaitGroup {
	return h.wg
}

func (h *WorkersHandle) Wait() {
	if h == nil || h.wg == nil {
		return
	}
	h.wg.Wait()
}

func (h *WorkersHandle) Shutdown(ctx context.Context) error {
	if h == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		h.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func StartWorkers(
	ctx context.Context,
	handle *WorkersHandle,
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

	events.StartFileNotificationBuffer(ctx, handle.wg, cache, notify)
	zap.L().Info("Started file notification buffer")

	if notificationsSub := eventsManager.GetSubscriber(configuration.EventsNotifications); notificationsSub != nil {
		notifications := notificationsSub.Subscribe()
		handle.wg.Go(func() {
			events.HandleEvents(ctx, eventParams, notifications)
		})
		zap.L().Info("Started notifications worker")
	}

	if RequiresUploadConfirmation(config.Storage.Type, config.Events.Type) {
		startWorker(ctx, handle.wg, profile.Workers.TrashCleanup, "trash_cleanup", cache, appIdentity,
			func(workerCtx context.Context) {
				worker := &workers.TrashCleanupWorker{
					DB:                 db,
					Publisher:          eventRouter,
					TrashRetentionDays: config.App.TrashRetentionDays,
					RunInterval:        time.Duration(config.App.TrashRetentionDays) * 24 * time.Hour / 7,
				}
				worker.Start(workerCtx)
			})
	}

	startWorker(ctx, handle.wg, profile.Workers.GarbageCollector, "garbage_collector", cache, appIdentity,
		func(workerCtx context.Context) {
			worker := &workers.GarbageCollectorWorker{
				DB:                 db,
				Storage:            store,
				Cache:              cache,
				ActivityLogger:     activityLogger,
				RunInterval:        15 * time.Minute,
				RefreshTokenExpiry: config.App.RefreshTokenExpiry,
			}
			worker.Start(workerCtx)
		})

	if deletionSub := eventsManager.GetSubscriber(configuration.EventsObjectDeletion); deletionSub != nil {
		deletionMessages := deletionSub.Subscribe()
		startWorker(ctx, handle.wg, profile.Workers.ObjectDeletion, "object_deletion", cache, appIdentity,
			func(workerCtx context.Context) {
				events.HandleEvents(workerCtx, eventParams, deletionMessages)
			})
	}

	if bucketSub := eventsManager.GetSubscriber(configuration.EventsBucketEvents); bucketSub != nil {
		bucketMessages := bucketSub.Subscribe()
		startWorker(ctx, handle.wg, profile.Workers.BucketEvents, "bucket_events", cache, appIdentity,
			func(workerCtx context.Context) {
				events.HandleBucketEvents(
					workerCtx,
					eventsManager.parser,
					db,
					activityLogger,
					store,
					eventRouter,
					config.App.TrashRetentionDays,
					bucketMessages,
				)
			})
	}
}

func startWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
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
		wg.Go(func() {
			startSingletonWorker(ctx, wg, cache, appIdentity, workerName, runWorker)
		})
	} else {
		wg.Go(func() { runWorker(ctx) })
		zap.L().Info("Started worker", zap.String("worker", workerName))
	}
}

func startSingletonWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	cache c.ICache,
	instanceID string,
	workerName string,
	runWorker func(context.Context),
) {
	lockKey := fmt.Sprintf(configuration.CacheAppWorkerLockKey, workerName)
	lockTTL := time.Duration(configuration.CacheAppWorkerLockTTL) * time.Second
	ticker := time.NewTicker(time.Duration(configuration.CacheAppWorkerLockRefresh) * time.Second)
	defer ticker.Stop()

	var workerStarted bool
	var cancelWorker context.CancelFunc
	var workerDone chan struct{}
	defer func() {
		if cancelWorker != nil {
			cancelWorker()
		}
		if workerDone != nil {
			<-workerDone
		}
	}()
	stopWorker := func() {
		if cancelWorker != nil {
			cancelWorker()
			cancelWorker = nil
		}
		if workerDone != nil {
			<-workerDone
			workerDone = nil
		}
		workerStarted = false
	}

	for {
		if !workerStarted {
			acquired, err := cache.SetNX(lockKey, instanceID, lockTTL)
			if err != nil {
				zap.L().Error("Failed to acquire worker lock", zap.String("worker", workerName), zap.Error(err))
			}

			if acquired {
				zap.L().Info("Acquired worker lock, starting worker", zap.String("worker", workerName))
				workerStarted = true
				workerCtx, cancel := context.WithCancel(ctx)
				cancelWorker = cancel
				done := make(chan struct{})
				workerDone = done
				wg.Go(func() {
					defer close(done)
					runWorker(workerCtx)
				})
			}
		} else {
			refreshed, err := c.RefreshLock(cache, lockKey, instanceID, lockTTL)
			if err != nil || !refreshed {
				zap.L().Warn("Lost worker lock, stopping worker", zap.String("worker", workerName))
				stopWorker()
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func StartIdentityTicker(ctx context.Context, wg *sync.WaitGroup, cache c.ICache, id string) {
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

	wg.Go(func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if err := register(); err != nil {
				zap.L().Error("App identity ticker failed to register", zap.Error(err))
			}
			if err := cleanup(); err != nil {
				zap.L().Error("App identity ticker failed to cleanup", zap.Error(err))
			}
		}
	})
}

func BuildAPIRouter(
	config models.Configuration,
	db *gorm.DB,
	cache c.ICache,
	store storage.IStorage,
	activityLogger activity.IActivityLogger,
	notify notifier.INotifier,
	publisher messaging.IPublisher,
	providers configuration.Providers,
) chi.Router {
	r := chi.NewRouter()

	if config.Tracing.Enabled {
		r.Use(otelhttp.NewMiddleware(
			configuration.AppName,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + r.URL.Path
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return strings.HasPrefix(r.URL.Path, "/api/")
			}),
		))
	}

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
			Publisher:          publisher,
			Notifier:           notify,
			ActivityLogger:     activityLogger,
			RefreshTokenExpiry: configuration.RefreshTokenExpiry,
		}

		apiRouter.Mount("/v1/users", userService.Routes())

		apiRouter.Mount("/v1/mfa", services.MFAService{
			DB:             db,
			Cache:          cache,
			AuthConfig:     authConfig,
			Publisher:      publisher,
			Notifier:       notify,
			ActivityLogger: activityLogger,
		}.Routes())

		apiRouter.Mount("/v1/buckets", services.BucketService{
			DB:                 db,
			Storage:            store,
			Publisher:          publisher,
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
			Publisher:      publisher,
			ActivityLogger: activityLogger,
		}.Routes())

		apiRouter.Mount("/v1/invites", services.InviteService{
			DB:             db,
			Cache:          cache,
			Storage:        store,
			AuthConfig:     authConfig,
			Publisher:      publisher,
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
			Publisher:      publisher,
			JWTSecret:      authConfig.JWTSecret,
		}.Routes())
	})

	return r
}

func StartHTTPServer(
	config models.Configuration,
	router chi.Router,
	embeddedWebFS embed.FS,
) (func(context.Context) error, <-chan error) {
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
		router.Mount("/", staticFileService.Routes())
		zap.L().Info("static file service enabled", zap.String("source", "embedded"))
	} else {
		zap.L().Info("static file service disabled")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.App.Port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
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
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("HTTP server exited", zap.Error(err))
			errCh <- err
		}
	}()

	return server.Shutdown, errCh
}
