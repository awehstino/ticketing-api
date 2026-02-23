package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using fallback defaults.")
	}

	// Connect to MySQL
	database.Connect()

	// Init Gin
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		// Get the underlying sql.DB object from GORM
		sqlDB, err := database.DB.DB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to get database instance"})
			return
		}

		// Ping the database with a timeout to check the connection
		ctx, cancel := context.WithTimeout(c, 1*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "unreachable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "connected"})
	})

	// Setup application routes
	routes.SetupRoutes(r)

	// Run server
	r.Run(":8080")
}
