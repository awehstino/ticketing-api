package controllers

import (
	"net/http"
	"strconv"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/gin-gonic/gin"
)

type AddStaffInput struct {
	Email string `json:"email" binding:"required,email"`
}

// helper: convert string → uint
func parseUint(s string) uint {
	v, _ := strconv.ParseUint(s, 10, 64)
	return uint(v)
}

// AddStaffToEvent - invite staff by email
func AddStaffToEvent(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("event_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var input AddStaffInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Create new minimal staff account
		user = models.User{
			Name:  input.Email,
			Email: input.Email,
			Role:  "attendee", // default role, but linked to event as staff
		}
		database.DB.Create(&user)
	}

	// Add staff to event
	staff := models.EventStaff{
		EventID: uint(eventID),
		UserID:  user.ID,
		Role:    "staff",
	}
	if err := database.DB.Create(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add staff"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff added", "staff": staff})
}

// GetScannerEvents returns a list of events a user can scan for, based on their role.
// - Admins get all events.
// - Organizers get events they created.
// - Staff get events they are assigned to.
func GetScannerEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	role := c.GetString("role")

	var events []models.Event
	var err error

	switch role {
	case "admin":
		err = database.DB.Preload("Category").Order("start_time desc").Find(&events).Error
	case "organizer":
		err = database.DB.Preload("Category").Where("organizer_id = ?", userID).Order("start_time desc").Find(&events).Error
	default: // "staff" or other roles
		// Find all event staff records for the user
		var staffAssignments []models.EventStaff
		database.DB.Where("user_id = ?", userID).Find(&staffAssignments)

		var eventIDs []uint
		for _, assignment := range staffAssignments {
			eventIDs = append(eventIDs, assignment.EventID)
		}
		err = database.DB.Preload("Category").Where("id IN ?", eventIDs).Order("start_time desc").Find(&events).Error
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch events"})
		return
	}

	eventResponses := make([]EventResponse, 0)
	for _, event := range events {
		eventResponses = append(eventResponses, toEventResponse(event))
	}
	c.JSON(http.StatusOK, gin.H{"events": eventResponses})
}

// ListEventStaff - list staff for event
func ListEventStaff(c *gin.Context) {
	eventID := c.Param("event_id")

	staff := make([]models.EventStaff, 0)
	if err := database.DB.Preload("User").Where("event_id = ?", eventID).Find(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": staff})
}

// RemoveStaffFromEvent - remove a staff member
func RemoveStaffFromEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	userID := c.Param("user_id")

	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).
		Delete(&models.EventStaff{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove staff"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff removed"})
}
