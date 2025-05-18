package main

import (
	c "api/internal/cache"
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
	s3 := core.InitStorage(config.Storage)
	mailer := core.NewMailer(config.Mailer)
	publisher := core.NewPublisher(config.Events, configuration.EventsNotificationsTopicName)
	activity := core.NewActivityLogger(config.Activity)
	subscriber := core.NewSubscriber(config.Events)
	messages := subscriber.Subscribe(context.Background(), configuration.EventsNotificationsTopicName)

	model := rbac.GetModel()
	a, _ := gormadapter.NewAdapterByDBWithCustomTable(db, &models.Policy{}, configuration.PolicyTableName)
	e, _ := casbin.NewEnforcer(model, a)

	_ = roles.InsertRoleGuest(e)
	_ = roles.InsertRoleUser(e)
	_ = roles.InsertRoleAdmin(e)

	// TODO: Create a dedicated fct

	adminUser := models.User{
		FirstName: "admin",
		LastName:  "admin",
		Email:     config.Admin.Username,
	}

	hash, _ := h.CreateHash(config.Admin.Password)
	adminUser.HashedPassword = hash
	db.Create(&adminUser)
	_ = roles.AddUserToRoleAdmin(e, adminUser)

	//

	appIdentity := uuid.New().String()

	go events.HandleNotifications(config.Platform.WebUrl, mailer, messages)

	go func() {
		err := cache.StartIdentityTicker(appIdentity)
		if err != nil {
			log.Fatalf("Platform identity ticker crashed: %v\n", err)
		}
	}()

	r := chi.NewRouter()

	r.Use(middleware.Timeout(5 * time.Second))
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

	r.Use(m.Authenticate(config.JWT))

	providers := configuration.LoadProviders(context.Background(), config.Platform.ApiUrl, config.Auth.Providers)

	r.Mount("/users", services.UserService{DB: db, Enforcer: e}.Routes())
	r.Mount("/buckets", services.BucketService{
		DB:             db,
		S3:             s3,
		Enforcer:       e,
		Publisher:      &publisher,
		ActivityLogger: activity,
	}.Routes())

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
