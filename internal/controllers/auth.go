package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"adibhaiya-balkan/internal/models"
	"adibhaiya-balkan/internal/utils"
)

type AuthController struct {
	db    *gorm.DB
	rdb   *redis.Client
	email *utils.SMTPClient
}

func NewAuthController(db *gorm.DB, rdb *redis.Client, email *utils.SMTPClient) *AuthController {
	return &AuthController{db: db, rdb: rdb, email: email}
}

// Signup: email + password (+ fullname)
type signupPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name"`
}

func (a *AuthController) SignUp(c *gin.Context) {
	var p signupPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.Email = strings.ToLower(p.Email)

	// hash
	hash, err := utils.HashPassword(p.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash"})
		return
	}
	user := models.User{Email: p.Email, Password: hash, FullName: p.FullName}
	if err := a.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}
	// send welcome email (non-blocking)
	go func() {
		if a.email != nil {
			_ = a.email.Send(user.Email, "Welcome", fmt.Sprintf("Hello %s,\n\nWelcome! Your account is created.", user.FullName))
		}
	}()
	c.JSON(http.StatusCreated, gin.H{"message": "signup successful"})
}

// Login step 1: verify email+password -> generate OTP -> save in redis -> send email
type loginPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (a *AuthController) Login(c *gin.Context) {
	var p loginPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email := strings.ToLower(p.Email)
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := utils.CheckPasswordHash(user.Password, p.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// generate OTP
	otp, err := utils.GenerateNumericOTP(6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate otp"})
		return
	}

	// store OTP in redis
	ttlMin := 5
	if v := os.Getenv("OTP_TTL_MIN"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			ttlMin = t
		}
	}
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", email)
	if err := a.rdb.Set(ctx, key, otp, time.Minute*time.Duration(ttlMin)).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store otp"})
		return
	}

	// send OTP email (non-blocking)
	go func() {
		if a.email != nil {
			body := fmt.Sprintf("Hello %s,\n\nYour OTP for login is: %s\nThis OTP will expire in %d minutes.\n\nIf you didn't request this, ignore.", user.FullName, otp, ttlMin)
			_ = a.email.Send(email, "Your login OTP", body)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "otp sent to email"})
}

// Verify OTP -> issue JWT
type verifyPayload struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

func (a *AuthController) VerifyOTP(c *gin.Context) {
	var p verifyPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email := strings.ToLower(p.Email)
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", email)
	stored, err := a.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "otp expired or not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
		return
	}
	if stored != p.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid otp"})
		return
	}

	// delete OTP once used
	_ = a.rdb.Del(ctx, key)

	// find user and issue JWT
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	tokenStr, err := createAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": tokenStr})
}

func createAccessToken(userID uint) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	mins := 60
	if v := os.Getenv("ACCESS_TOKEN_EXPIRES_MIN"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			mins = t
		}
	}
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Minute * time.Duration(mins)).Unix(),
		"typ": "access",
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

// Protected route
func (a *AuthController) Me(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no user in context"})
		return
	}
	var user models.User
	if err := a.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": user.ID, "email": user.Email, "full_name": user.FullName}})
}
