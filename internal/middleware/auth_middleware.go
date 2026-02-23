package middleware

import (
	"net/http"
	"strings"

	
	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/awehstino/ticketing-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

       token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
          return utils.JwtSecret, nil
       })
        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
            c.Abort()
            return
        }

        userIDFloat, ok := claims["user_id"].(float64)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token user ID"})
            c.Abort()
            return
        }
        userID := uint(userIDFloat)

        var user models.User
        if err := database.DB.First(&user, userID).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
            c.Abort()
            return
        }

        // Ensure Role and AccountType are strings
        c.Set("userID", user.ID)
        c.Set("role", string(user.Role))
        c.Set("account_type", string(user.AccountType))

        c.Next()
    }
}

func OrganizerOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        roleVal, _ := c.Get("role")
        accountTypeVal, _ := c.Get("account_type")

        role, ok1 := roleVal.(string)
        accountType, ok2 := accountTypeVal.(string)

        if !ok1 || !ok2 {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
            c.Abort()
            return
        }

        if role != "admin" && accountType != "business" {
            c.JSON(http.StatusForbidden, gin.H{"error": "Only business accounts can perform this action"})
            c.Abort()
            return
        }

        c.Next()
    }
}
