package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/gin-gonic/gin"
)

type CheckinInput struct {
	TicketCode string `json:"ticket_code" binding:"required"`
}

// RevokeCheckIn - undo check-in for a ticket within event
func RevokeCheckIn(c *gin.Context) {
	eventIDParam := c.Param("event_id")
	eventID, err := strconv.ParseUint(eventIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	ticketCode := c.Param("ticket_code")

	// Use a transaction to ensure atomicity
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	// Defer a rollback in case of panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var item models.OrderItem
	if err := tx.Preload("Ticket").Where("ticket_code = ?", ticketCode).First(&item).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Ensure ticket belongs to this event
	if item.Ticket.EventID != uint(eventID) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket does not belong to this event"})
		return
	}

	// Ensure ticket was actually checked in
	if !item.CheckedIn {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket has not been checked in yet"})
		return
	}

	// Revoke the check-in status on the order item
	item.CheckedIn = false
	item.CheckedInAt = nil
	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke check-in"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Check-in revoked successfully",
		"ticket_code": item.TicketCode,
		"event_id":    eventID,
	})

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
}

// CheckInTicket - validate ticket code by event and mark checked_in
func CheckInTicket(c *gin.Context) {
	var input CheckinInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse eventID from URL
	eventIDParam := c.Param("event_id")
	eventID, err := strconv.ParseUint(eventIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Load order item with ticket + order
	var item models.OrderItem
	if err := database.DB.
		Preload("Ticket").
		Preload("Order.User").
		Preload("Order.Guest").
		Where("ticket_code = ?", input.TicketCode).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Ensure ticket belongs to this event
	if item.Ticket.EventID != uint(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket does not belong to this event"})
		return
	}

	// Already checked in?
	if item.CheckedIn {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket already checked in"})
		return
	}

	// Update check-in
	item.CheckedIn = true
	now := time.Now()
	item.CheckedInAt = &now
	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket status"})
		return
	}

	// Get attendee info (User or Guest)
	var name, email string
	if item.Order.User != nil {
		name = item.Order.User.Name
		email = item.Order.User.Email
	} else if item.Order.Guest != nil {
		name = item.Order.Guest.Name
		email = item.Order.Guest.Email
	}

	// Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket checked in successfully",
		"attendee": gin.H{
			"ticket_code":   item.TicketCode,
			"ticket_name":   item.Ticket.Name,
			"checked_in":    item.CheckedIn,
			"checked_in_at": item.CheckedInAt,
			"price":         item.UnitPrice,
			"name":          name,
			"email":         email,
		},
	})
}
