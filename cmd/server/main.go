package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"adibhaiya-balkan/internal/controllers"
	"adibhaiya-balkan/internal/db"
	"adibhaiya-balkan/internal/middleware"
	"adibhaiya-balkan/internal/redis"
	"adibhaiya-balkan/internal/utils"
)

func main() {

	dbConn := db.Init()

	rdb := redis.Init()

	email := utils.NewSMTPClient(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
		os.Getenv("FROM_EMAIL"),
	)

	r := gin.Default()

	auth := controllers.NewAuthController(dbConn, rdb, email)

	api := r.Group("/api")
	{
		api.POST("/signup", auth.SignUp)
		api.POST("/login", auth.Login)          // step 1: password -> send OTP
		api.POST("/verify-otp", auth.VerifyOTP) // step 2: otp -> issue JWT
	}

	protected := r.Group("/api")
	protected.Use(middleware.JWTMiddleware(os.Getenv("JWT_SECRET")))
	{
		protected.GET("/me", auth.Me)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
	for _, route := range r.Routes() {
		fmt.Println(route.Method, route.Path)
	}

}
