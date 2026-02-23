package middleware

import (
	"net/http"
	"strconv"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/gin-gonic/gin"
)

// RequireEventStaffOrRole - organizers/admins OR staff for event
func RequireEventStaffOrRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userID, ok := uid.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		role := c.GetString("role")

		eventIDStr := c.Param("event_id")
		eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
			c.Abort()
			return
		}

		// Organizers/Admins → allow
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		// Staff check (only for this event)
		var count int64
		database.DB.Model(&models.EventStaff{}).
			Where("event_id = ? AND user_id = ?", eventID, userID).
			Count(&count)

		if count > 0 {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		c.Abort()
	}
}

// RequireEventStaffOnly - restricts to assigned staff (no admins/organizers)
func RequireEventStaffOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userID, ok := uid.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		eventIDStr := c.Param("event_id")
		eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
			c.Abort()
			return
		}

		// Only allow if user is staff for this event
		var count int64
		database.DB.Model(&models.EventStaff{}).
			Where("event_id = ? AND user_id = ?", eventID, userID).
			Count(&count)

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Staff access only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
