package middleware

import (
	"strings"

	"github.com/awehstino/ticketing-api/internal/config"
	"github.com/awehstino/ticketing-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// OptionalAuthMiddleware tries to authenticate a user if a token is provided,
// but does not fail if the token is missing. This is for routes that serve
// both authenticated users and guests.
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next() // No token, proceed as guest
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next() // Malformed header, proceed as guest
			return
		}

		claims := &utils.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return config.AppJWTConfig.SecretKey, nil
		})

		// If token is valid, set user info in context
		if err == nil && token.Valid {
			c.Set("userID", claims.UserID)
			c.Set("role", claims.Role)
		}

		c.Next()
	}
}
