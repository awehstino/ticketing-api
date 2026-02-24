package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/awehstino/ticketing-api/internal/config"
	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/awehstino/ticketing-api/internal/utils"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

// EventResponse defines the public-facing structure for an event, including a full URL for the image.
type EventResponse struct {
	ID          uint            `json:"id"`
	OrganizerID uint            `json:"organizer_id"`
	CategoryID  uint            `json:"category_id"`
	Title       string          `json:"title"`
	EventImage  string          `json:"event_image_url"`
	EventType   string          `json:"event_type"`
	Status      string          `json:"status"`
	Location    string          `json:"location"`
	Description string          `json:"description"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	Venue       string          `json:"venue"`
	CreatedAt   time.Time       `json:"created_at"`
	Category    models.Category `json:"category"`
}

// toEventResponse converts an Event model to a public-facing EventResponse DTO.
func toEventResponse(event models.Event) EventResponse {
	imageURL := ""
	if event.EventImage != "" {
		// The path is stored relative to the project root, e.g., "public\uploads\events\image.png".
		// We need to convert backslashes to forward slashes for the URL.
		imageURL = config.AppBaseURL + "/" + filepath.ToSlash(event.EventImage)
	}
	return EventResponse{
		ID:          event.ID,
		OrganizerID: event.OrganizerID,
		CategoryID:  event.CategoryID,
		Title:       event.Title,
		EventImage:  imageURL,
		EventType:   event.EventType,
		Status:      event.Status,
		Location:    event.Location,
		Description: event.Description,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		Venue:       event.Venue,
		CreatedAt:   event.CreatedAt,
		Category:    event.Category,
	}
}

type CreateEventInput struct {
	Title       string    `form:"title" binding:"required"`
	Location    string    `form:"location" binding:"required"`
	Description string    `form:"description"`
	CategoryID  uint      `form:"category_id" binding:"required"`
	EventType   string    `form:"event_type"`
	// EventImage is handled as a file upload, not in this struct
	StartTime time.Time `form:"start_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime   time.Time `form:"end_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	Venue     string    `form:"venue"`
}

// ListTickets - fetch all tickets for an event
func ListTickets(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	tickets := make([]models.Ticket, 0)
	if err := database.DB.Where("event_id = ?", eventID).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

// ListEvents - fetch all events
func ListEvents(c *gin.Context) {
	var events []models.Event
	if err := database.DB.Preload("Category").Where("status = ?", "publish").Order("start_time desc").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	eventResponses := make([]EventResponse, 0)
	for _, event := range events {
		eventResponses = append(eventResponses, toEventResponse(event))
	}
	c.JSON(http.StatusOK, gin.H{"events": eventResponses})
}

// ListMyEvents - fetch all events created by the authenticated organizer for their dashboard
func ListMyEvents(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var events []models.Event
	// Fetch all events for the organizer, regardless of status
	if err := database.DB.Preload("Category").Where("organizer_id = ?", userID).Order("created_at desc").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your events"})
		return
	}

	eventResponses := make([]EventResponse, 0)
	for _, event := range events {
		eventResponses = append(eventResponses, toEventResponse(event))
	}

	c.JSON(http.StatusOK, gin.H{"events": eventResponses})
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

	c.JSON(http.StatusOK, gin.H{"event": toEventResponse(event)})
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
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle file upload (optional on update)
	file, err := c.FormFile("event_image")
	if err == nil {
		// A new file was provided, process it
		filename := uuid.New().String() + filepath.Ext(file.Filename)
		// Relative path for DB and URL construction
		relativeImagePath := filepath.Join("public", "uploads", "events", filename)
		// Absolute path for saving the file to disk
		absoluteImagePath := filepath.Join(utils.ProjectRoot, relativeImagePath)

		// Ensure the absolute directory exists
		if err := os.MkdirAll(filepath.Dir(absoluteImagePath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		log.Printf("Attempting to save updated file to: %s", absoluteImagePath)
		if err := c.SaveUploadedFile(file, absoluteImagePath); err != nil {
			log.Printf("ERROR: Failed to save updated file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image", "detail": err.Error()})
			return
		}
		// If there was an old image, delete it from storage
		if event.EventImage != "" {
			// Construct absolute path to old image for deletion
			os.Remove(filepath.Join(utils.ProjectRoot, event.EventImage))
		}
		event.EventImage = relativeImagePath // Save the relative path
	}

	// Map input to the event model
	event.Title = input.Title
	event.Location = input.Location
	event.Description = input.Description
	event.CategoryID = input.CategoryID
	event.EventType = input.EventType
	event.StartTime = input.StartTime
	event.EndTime = input.EndTime
	event.Venue = input.Venue

	database.DB.Save(&event)
	database.DB.Preload("Category").First(&event, event.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Event updated successfully",
		"event":   toEventResponse(event),
	})
}

func CreateEvent(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input CreateEventInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle file upload
	var relativeImagePath string
	file, err := c.FormFile("event_image")
	if err == nil {
		// A file was provided
		// Generate a unique filename to prevent overwrites
		filename := uuid.New().String() + filepath.Ext(file.Filename)

		// Relative path for DB and URL construction
		relativeImagePath = filepath.Join("public", "uploads", "events", filename)
		// Absolute path for saving the file to disk
		absoluteImagePath := filepath.Join(utils.ProjectRoot, relativeImagePath)

		// Ensure the absolute directory exists
		if err := os.MkdirAll(filepath.Dir(absoluteImagePath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		// Save the file
		log.Printf("Attempting to save new file to: %s", absoluteImagePath)
		if err := c.SaveUploadedFile(file, absoluteImagePath); err != nil {
			log.Printf("ERROR: Failed to save new file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image", "detail": err.Error()})
			return
		}
	}
	// Map input to the event model
	event := models.Event{
		OrganizerID: userID,
		Title:       input.Title,
		Location:    input.Location,
		Description: input.Description,
		CategoryID:  input.CategoryID,
		EventType:   input.EventType,
		EventImage:  relativeImagePath, // Use the saved relative path
		Status:      "draft", // Default to draft until tickets are created
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
		"event":   toEventResponse(event),
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

// UpdateEventStatus updates the status of an event (e.g., to 'publish').
func UpdateEventStatus(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var input struct {
		Status string `json:"status" form:"status" binding:"required,oneof=publish draft pending"`
	}
	if err := c.ShouldBind(&input); err != nil {
		log.Printf("UpdateEventStatus bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Authorization check
	if event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to update this event's status"})
		return
	}

	// Logic check: can't publish an event with no tickets
	// Commented out to allow publishing events without tickets (e.g. free/info events)
	// if input.Status == "publish" {
	// 	var ticketCount int64
	// 	database.DB.Model(&models.Ticket{}).Where("event_id = ?", eventID).Count(&ticketCount)
	// 	if ticketCount == 0 {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot publish an event with no tickets."})
	// 		return
	// 	}
	// }

	if err := database.DB.Model(&event).Update("status", input.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event status"})
		return
	}

	database.DB.Preload("Category").First(&event, event.ID)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Event status updated to '%s'", input.Status), "event": toEventResponse(event)})
}
