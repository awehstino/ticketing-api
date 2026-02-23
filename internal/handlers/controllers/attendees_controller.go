package controllers

import (
	"net/http"
	"strconv"

	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/gin-gonic/gin"
)

type Attendee struct {
	TicketCode string  `json:"ticket_code"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	CheckedIn  bool    `json:"checked_in"`
	TicketName string  `json:"ticket_name"`
	Price      float64 `json:"price"`
}

// GetAttendeesReport - list attendees for an event
func GetAttendeesReport(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var items []models.OrderItem
	if err := database.DB.
		Preload("Order.User").
		Preload("Order.Guest").
		Preload("Ticket").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.event_id = ?", eventID).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attendees"})
		return
	}

	var attendees []Attendee
	for _, item := range items {
		var name, email string

		if item.Order.User != nil {
			name = item.Order.User.Name
			email = item.Order.User.Email
		} else if item.Order.Guest != nil {
			name = item.Order.Guest.Name
			email = item.Order.Guest.Email
		}

		attendees = append(attendees, Attendee{
			TicketCode: item.TicketCode,
			Name:       name,
			Email:      email,
			CheckedIn:  item.CheckedIn,
			TicketName: item.Ticket.Name,
			Price:      item.UnitPrice,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":  eventID,
		"attendees": attendees,
	})
}
