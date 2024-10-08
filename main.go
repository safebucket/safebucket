package main

import (
	"api/internal/configuration"
	"api/internal/database"
	"api/internal/models"
	"api/internal/services"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	config := configuration.Read()
	db := database.InitDB(config.Database)

	err := db.AutoMigrate(&models.User{}, &models.Bucket{}, &models.File{})
	if err != nil {
		zap.L().Error("failed to migrate db models", zap.Error(err))
	}

	r := chi.NewRouter()

	// A good base middleware stack
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

	ctx := context.Background()
	providers := configuration.LoadProviders(ctx, config.Platform.ApiUrl, config.Auth.Providers)

	r.Mount("/users", services.UserService{DB: db}.Routes())
	r.Mount("/buckets", services.BucketService{DB: db}.Routes())
	r.Mount("/auth", services.AuthService{
		DB:        db,
		JWTConf:   config.JWT,
		Providers: providers,
		WebUrl:    config.Platform.WebUrl,
	}.Routes())

	zap.L().Info("App started")

	err = http.ListenAndServe(":1323", r)
	if err != nil {
		zap.L().Error("Failed to start the app")
	}
}
