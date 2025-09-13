package main

import (
	"api/internal/configuration"
	"api/internal/core"
	"api/internal/database"
	"api/internal/events"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/roles"
	"api/internal/services"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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
	storage := core.NewStorage(config.Storage)
	notifier := core.NewNotifier(config.Notifier)
	publisher := core.NewPublisher(config.Events)
	activity := core.NewActivityLogger(config.Activity)

	notificationsSubscriber := core.NewSubscriber(config.Events)
	notifications := notificationsSubscriber.Subscribe()

	bucketEventsSubscriber := core.NewBucketEventsSubscriber(config.Storage, storage)
	bucketEvents := bucketEventsSubscriber.Subscribe()

	model := rbac.GetModel()
	a, _ := gormadapter.NewAdapterByDBWithCustomTable(db, &models.Policy{}, configuration.PolicyTableName)
	enforcer, _ := casbin.NewEnforcer(model, a)

	_ = roles.InsertRoleGuest(enforcer)
	_ = roles.InsertRoleUser(enforcer)
	_ = roles.InsertRoleAdmin(enforcer)

	// TODO: Create a dedicated fct

	adminUser := models.User{
		FirstName: "admin",
		LastName:  "admin",
		Email:     config.App.AdminEmail,
	}

	hash, _ := h.CreateHash(config.App.AdminPassword)
	adminUser.HashedPassword = hash
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"hashed_password"}),
	}).Create(&adminUser)
	_ = roles.AddUserToRoleAdmin(enforcer, adminUser)

	//

	appIdentity := uuid.New().String()

	go events.HandleNotifications(config.App.WebUrl, notifier, notifications)

	go events.HandleBucketEvents(bucketEventsSubscriber, db, activity, bucketEvents)

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

	providers := configuration.LoadProviders(context.Background(), config.App.ApiUrl, config.Auth.Providers)

	// API routes with auth middleware
	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(m.Authenticate(config.App.JWTSecret))
		apiRouter.Use(m.RateLimit(cache, config.App.TrustedProxies))

		apiRouter.Mount("/v1/users", services.UserService{DB: db, Enforcer: enforcer}.Routes())
		apiRouter.Mount("/v1/buckets", services.BucketService{
			DB:             db,
			Storage:        storage,
			Enforcer:       enforcer,
			Publisher:      &publisher,
			ActivityLogger: activity,
			Providers:      providers,
			WebUrl:         config.App.WebUrl,
		}.Routes())

		apiRouter.Mount("/v1/auth", services.AuthService{
			DB:        db,
			Enforcer:  enforcer,
			JWTSecret: config.App.JWTSecret,
			Providers: providers,
			WebUrl:    config.App.WebUrl,
		}.Routes())

		apiRouter.Mount("/v1/invites", services.InviteService{
			DB:             db,
			JWTSecret:      config.App.JWTSecret,
			Enforcer:       enforcer,
			Publisher:      &publisher,
			ActivityLogger: activity,
			Providers:      providers,
			WebUrl:         config.App.WebUrl,
		}.Routes())
	})

	// Initialize and mount static file service (if enabled)
	if config.App.StaticFiles.Enabled {
		staticFileService, err := services.NewStaticFileService(
			config.App.StaticFiles.Directory,
			config.App.ApiUrl,
		)
		if err != nil {
			zap.L().Fatal("failed to initialize static file service", zap.Error(err))
		}
		r.Mount("/", staticFileService.Routes())
		zap.L().Info("static file service enabled", zap.String("directory", config.App.StaticFiles.Directory))
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
