package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateEventInput struct {
	Title       string    `json:"title" binding:"required"`
	Location    string    `json:"location" binding:"required"`
	Description string    `json:"description"`
	CategoryID  uint      `json:"category_id" binding:"required"`
	EventType   string    `json:"event_type"`
	EventImage  string    `json:"event_image"`
	StartTime   time.Time `json:"start_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime     time.Time `json:"end_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	Venue       string    `json:"venue"`
}

// ListTickets - fetch all tickets for an event
func ListTickets(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var tickets []models.Ticket
	if err := database.DB.Where("event_id = ?", eventID).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

// ListEvents - fetch all events
func ListEvents(c *gin.Context) {
	var events []models.Event
	if err := database.DB.Preload("Category").Order("start_time desc").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetEvent - fetch a single event by ID
func GetEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var event models.Event
	if err := database.DB.Preload("Category").First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"event": event})
}

// UpdateEvent - update an existing event
func UpdateEvent(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Authorization check
	if event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to update this event"})
		return
	}

	var input CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map input to the event model
	event.Title = input.Title
	event.Location = input.Location
	event.Description = input.Description
	event.CategoryID = input.CategoryID
	event.EventType = input.EventType
	event.EventImage = input.EventImage
	event.StartTime = input.StartTime
	event.EndTime = input.EndTime
	event.Venue = input.Venue

	database.DB.Save(&event)

	c.JSON(http.StatusOK, gin.H{
		"message": "Event updated successfully",
		"event":   event,
	})
}

func CreateEvent(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map input to the event model
	event := models.Event{
		OrganizerID: userID,
		Title:       input.Title,
		Location:    input.Location,
		Description: input.Description,
		CategoryID:  input.CategoryID,
		EventType:   input.EventType,
		EventImage:  input.EventImage,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Venue:       input.Venue,
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to create event",
			"detail": err.Error(),
		})
		return
	}
	// Preload the category to return it in the response
	database.DB.Preload("Category").First(&event, event.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Event created successfully",
		"event":   event,
	})
}

// DeleteEvent - delete an event
func DeleteEvent(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Authorization check
	if event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to delete this event"})
		return
	}

	database.DB.Delete(&event)

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
}
