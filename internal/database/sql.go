package database

import (
	"api/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(config models.DatabaseConfiguration) *gorm.DB {
	dsn := "host=" + config.Host + "user=" + config.User + "password=" + config.Password + "dbname=" + config.Name + "port=" + string(config.Port) + "sslmode=" + config.SSLMode
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Error("Failed to connect to database", zap.Error(err))
	}
	return db
}
