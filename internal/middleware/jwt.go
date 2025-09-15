package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(secret string) gin.HandlerFunc {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokStr := parts[1]
		tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenMalformed
			}
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := tok.Claims.(jwt.MapClaims)
		if claims["typ"] != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not an access token"})
			return
		}
		// sub can be float64 because of JSON encoding; handle carefully
		sub := claims["sub"]
		switch v := sub.(type) {
		case float64:
			c.Set("user_id", uint(v))
		case int:
			c.Set("user_id", uint(v))
		case int64:
			c.Set("user_id", uint(v))
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}
		c.Next()
	}
}
