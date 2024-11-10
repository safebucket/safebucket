package main

import (
	"api/internal/authorization"
	c "api/internal/cache"
	"api/internal/configuration"
	"api/internal/database"
	"api/internal/models"
	"api/internal/services"
	"context"
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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

	// Casbin
	model := authorization.GetModel()
	a, _ := gormadapter.NewAdapterByDBWithCustomTable(db, &models.Policies{}, configuration.PolicyTableName)
	e, _ := casbin.NewEnforcer(model, a)
	e.AddPolicy("d4f06f25-7fa5-44f7-9211-ae8b1bbe9c0b", "1", "d4f06f25-7fa5-44f7-9211-ae8b1bbe9c0a", "read")
	data, _ := e.Enforce("d4f06f25-7fa5-44f7-9211-ae8b1bbe9c0b", "1", "d4f06f25-7fa5-44f7-9211-ae8b1bbe9c0a", "read")

	data2, _ := e.GetFilteredPolicy(0, "d4f06f25-7fa5-44f7-9211-ae8b1bbe9c0b", "2", "delete")
	fmt.Printf("%v", data2)
	//zap.L().Info(data2)
	if data {
		zap.L().Info("OK")
	} else {
		zap.L().Info("KO")
	}
	// End

	err := db.AutoMigrate(&models.User{}, &models.Bucket{}, &models.File{})
	if err != nil {
		zap.L().Error("failed to migrate db models", zap.Error(err))
	}

	appIdentity := uuid.New().String()

	go func() {
		err := cache.StartIdentityTicker(appIdentity)
		if err != nil {
			log.Fatalf("Platform identity ticker crashed: %v\n", err)
		}
	}()

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
