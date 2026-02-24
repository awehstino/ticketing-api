package controllers

import (
	"net/http"
	"strconv"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/gin-gonic/gin"
)

type CreateTicketInput struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Type        string  `json:"type" binding:"required,oneof=free paid"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity" binding:"required"`
}

// CreateTicket - add ticket type to an event
func CreateTicket(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Check if the authenticated user is the organizer of this event
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	if event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to add tickets to this event"})
		return
	}

	var input CreateTicketInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If ticket is free, ensure price is 0
	if input.Type == "free" {
		input.Price = 0
	}

	ticket := models.Ticket{
		EventID:     uint(eventID),
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		Price:       input.Price,
		Quantity:    input.Quantity,
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}

	// Update event status to 'pending' if it's in 'draft'
	if event.Status == "draft" {
		database.DB.Model(&event).Update("status", "pending")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket created successfully", "ticket": ticket})
}

// GetTicket - fetch a single ticket by ID
func GetTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := database.DB.Preload("Event").First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

// UpdateTicket - update an existing ticket
func UpdateTicket(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := database.DB.Preload("Event").First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Authorization check: user must be the organizer of the event this ticket belongs to
	if ticket.Event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to update this ticket"})
		return
	}

	var input CreateTicketInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If ticket is free, ensure price is 0
	if input.Type == "free" {
		input.Price = 0
	}

	ticket.Name = input.Name
	ticket.Description = input.Description
	ticket.Type = input.Type
	ticket.Price = input.Price
	ticket.Quantity = input.Quantity

	database.DB.Save(&ticket)

	c.JSON(http.StatusOK, gin.H{"message": "Ticket updated successfully", "ticket": ticket})
}

// DeleteTicket - delete a ticket
func DeleteTicket(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := database.DB.Preload("Event").First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Authorization check
	if ticket.Event.OrganizerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to delete this ticket"})
		return
	}

	// Prevent deletion if tickets have been sold
	var orderItemCount int64
	database.DB.Model(&models.OrderItem{}).Where("ticket_id = ?", ticket.ID).Count(&orderItemCount)
	if orderItemCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete ticket type with existing orders. Consider deactivating it instead."})
		return
	}

	database.DB.Delete(&ticket)

	c.JSON(http.StatusOK, gin.H{"message": "Ticket deleted successfully"})
}
