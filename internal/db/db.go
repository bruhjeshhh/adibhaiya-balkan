package db

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"adibhaiya-balkan/internal/models"
)

func Init() *gorm.DB {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db:", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("auto migrate failed:", err)
	}
	return db
}
