package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"api/internal/configuration"
	"api/internal/core"
	"api/internal/database"
	"api/internal/events"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	config := configuration.Read()
	core.NewLogger(config.App.LogLevel)

	db := database.InitDB(config.Database)
	cache := core.NewCache(config.Cache)
	storage := core.NewStorage(config.Storage, config.App.TrashRetentionDays)
	notifier := core.NewNotifier(config.Notifier)
	activity := core.NewActivityLogger(config.Activity)

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

	appIdentity := uuid.New().String()

	eventsManager := core.NewEventsManager(config.Events, storage)
	eventRouter := core.NewEventRouter(eventsManager)

	eventParams := &events.EventParams{
		WebURL:             config.App.WebURL,
		Notifier:           notifier,
		Publisher:          eventRouter,
		DB:                 db,
		Storage:            storage,
		ActivityLogger:     activity,
		TrashRetentionDays: config.App.TrashRetentionDays,
	}

	notifications := eventsManager.GetSubscriber(configuration.EventsNotifications).Subscribe()
	go events.HandleEvents(eventParams, notifications)

	deletionEvents := eventsManager.GetSubscriber(configuration.EventsObjectDeletion).Subscribe()
	go events.HandleEvents(eventParams, deletionEvents)

	bucketEventsSubscriber := eventsManager.GetSubscriber(configuration.EventsBucketEvents)
	bucketEvents := bucketEventsSubscriber.Subscribe()

	go events.HandleBucketEvents(
		bucketEventsSubscriber,
		db,
		activity,
		storage,
		config.App.TrashRetentionDays,
		bucketEvents,
	)

	go cache.StartIdentityTicker(appIdentity)

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
			Notifier:   notifier,
		}

		apiRouter.Mount("/v1/users", userService.Routes())

		apiRouter.Mount("/v1/mfa", services.MFAService{
			DB:             db,
			Cache:          cache,
			AuthConfig:     authConfig,
			Publisher:      eventRouter,
			Notifier:       notifier,
			ActivityLogger: activity,
		}.Routes())

		apiRouter.Mount("/v1/buckets", services.BucketService{
			DB:                 db,
			Storage:            storage,
			Publisher:          eventRouter,
			ActivityLogger:     activity,
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
			ActivityLogger: activity,
		}.Routes())

		apiRouter.Mount("/v1/invites", services.InviteService{
			DB:             db,
			Storage:        storage,
			AuthConfig:     authConfig,
			Publisher:      eventRouter,
			ActivityLogger: activity,
			Providers:      providers,
		}.Routes())

		apiRouter.Mount("/v1/admin", services.AdminService{
			DB:             db,
			ActivityLogger: activity,
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
		zap.L().
			Info("static file service enabled", zap.String("directory", config.App.StaticFiles.Directory))
	} else {
		zap.L().Info("static file service disabled")
	}

	zap.L().Info("App started")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.App.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		zap.L().Error("Failed to start the app")
	}
}
