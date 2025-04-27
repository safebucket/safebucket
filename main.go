package main

import (
	c "api/internal/cache"
	"api/internal/configuration"
	"api/internal/core"
	"api/internal/database"
	"api/internal/events"
	"api/internal/services"
	"api/internal/storage"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	config := configuration.Read()
	db := database.InitDB(config.Database)
	cache := c.InitCache(config.Redis)
	s3 := storage.InitStorage(config.Storage)
	mailer := core.NewMailer(config.Mailer)
	publisher := core.NewPublisher(config.Events, configuration.EventsNotificationsTopicName)
	subscriber := core.NewSubscriber(config.Events)
	messages := subscriber.Subscribe(context.Background(), configuration.EventsNotificationsTopicName)

	go events.HandleNotifications(config.Platform.WebUrl, mailer, messages)

	appIdentity := uuid.New().String()
	go func() {
		err := cache.StartIdentityTicker(appIdentity)
		if err != nil {
			log.Fatalf("Platform identity ticker crashed: %v\n", err)
		}
	}()

	r := chi.NewRouter()

	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.Cors.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	providers := configuration.LoadProviders(context.Background(), config.Platform.ApiUrl, config.Auth.Providers)

	r.Mount("/users", services.UserService{DB: db}.Routes())
	r.Mount("/buckets", services.BucketService{DB: db, S3: s3, Publisher: &publisher}.Routes())

	r.Mount("/auth", services.AuthService{
		DB:        db,
		JWTConf:   config.JWT,
		Providers: providers,
		WebUrl:    config.Platform.WebUrl,
	}.Routes())

	zap.L().Info("App started")

	server := &http.Server{
		Addr:         ":1323",
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
