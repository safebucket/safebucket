package main

import (
	"api/internal/configuration"
	"api/internal/database"
	"api/internal/models"
	"api/internal/repositories"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	config := configuration.Read()
	db := database.InitDB(config.Database)

	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Bucket{})
	db.AutoMigrate(&models.File{})

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", homePage)

	r.Mount("/users", repositories.UserRepo{DB: db}.Routes())
	r.Mount("/buckets", repositories.BucketRepo{DB: db}.Routes())

	zap.L().Info("App started")

	err := http.ListenAndServe(":1323", r)
	if err != nil {
		zap.L().Error("Failed to start the app")
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Welcome !")
	if err != nil {
		zap.L().Error("Failed to print Welcome")
	}
}
