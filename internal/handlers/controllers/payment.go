package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/awehstino/ticketing-api/internal/config"
	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/awehstino/ticketing-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type CheckoutInput struct {
	Guest *struct {
		Name  string `json:"name"`
		Email string `json:"email" binding:"required,email"`
		Phone string `json:"phone"`
	} `json:"guest"`
	Tickets []struct {
		TicketID uint `json:"ticket_id" binding:"required"`
		Quantity int  `json:"quantity" binding:"required,min=1"`
	} `json:"tickets" binding:"required,dive"`
}

type BudPayInitResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

// ✅ Checkout initializes BudPay payment and creates order + payment record
func Checkout(c *gin.Context) {
	var input CheckoutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID *uint
	var guestID *uint
	var email string

	if uid, exists := c.Get("userID"); exists {
		id := uid.(uint)
		userID = &id
		var user models.User
		database.DB.First(&user, id)
		email = user.Email
	} else if input.Guest != nil {
		var guest models.Guest
		if err := database.DB.Where("email = ?", input.Guest.Email).First(&guest).Error; err != nil {
			guest = models.Guest{
				Name:  input.Guest.Name,
				Email: input.Guest.Email,
				Phone: input.Guest.Phone,
			}
			database.DB.Create(&guest)
		}
		guestID = &guest.ID
		email = guest.Email
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User or guest info required"})
		return
	}

	// Build order items & total
	var total float64
	var items []models.OrderItem
	var eventID uint
	for _, t := range input.Tickets {
		var ticket models.Ticket
		if err := database.DB.First(&ticket, t.TicketID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
			return
		}

		// Check if the requested quantity exceeds the remaining quantity.
		if t.Quantity > ticket.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Not enough tickets available for '%s'. Only %d remaining.", ticket.Name, ticket.Quantity)})
			return
		}

		total += float64(t.Quantity) * ticket.Price
		items = append(items, models.OrderItem{
			TicketID:   t.TicketID,
			Quantity:   t.Quantity,
			UnitPrice:  ticket.Price,
			TotalPrice: float64(t.Quantity) * ticket.Price,
		})
		eventID = ticket.EventID
	}

	// Use a transaction to ensure atomicity
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create order
	order := models.Order{
		UserID:      userID,
		GuestID:     guestID,
		EventID:     eventID,
		TotalAmount: total,
		Status:      "pending",
		Items:       items,
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Create payment record
	ref := uuid.New().String()
	payment := models.Payment{
		OrderID:   order.ID,
		UserID:    userID,
		GuestID:   guestID,
		EventID:   eventID,
		Amount:    total,
		Status:    "pending",
		Reference: ref,
		CreatedAt: time.Now(),
	}
	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment record"})
		return
	}

	// If all good, commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// If total is 0, it's a free order. Fulfill it directly and skip payment gateway.
	if total == 0 {
		// Commit the initial transaction for order creation
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction for free order"})
			return
		}

		// Process the free order (generates tickets, sends email)
		finalOrder, err := processFreeOrder(order.ID)
		if err != nil {
			fmt.Printf("Error processing free order %d: %v\n", order.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process free order"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Free order processed successfully. Tickets have been sent to your email.",
			"order_id": finalOrder.ID,
			"order":    finalOrder,
		})
		return
	}

	// Initialize BudPay payment
	payload := map[string]interface{}{
		"email":     email,
		"amount":    fmt.Sprintf("%.0f", total*100),
		"reference": ref,
		"callback":  config.AppBaseURL + "/payments/confirm",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", config.BudPayBaseURL+"/transaction/initialize", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+config.BudPaySecretKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize payment"})
		return
	}
	defer resp.Body.Close()

	var budResp BudPayInitResponse
	json.NewDecoder(resp.Body).Decode(&budResp)

	if !budResp.Status {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BudPay returned an error", "message": budResp.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":    order.ID,
		"payment_ref": ref,
		"payment_url": budResp.Data.AuthorizationURL,
	})
}

// processFreeOrder handles the fulfillment of orders that have a total of 0.
// It marks the order as paid and generates/sends the tickets without payment verification.
func processFreeOrder(orderID uint) (*models.Order, error) {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find the order and related data
	var order models.Order
	if err := tx.Preload("User").Preload("Guest").Preload("Event").Preload("Items.Ticket").First(&order, orderID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("order not found: %d", orderID)
	}

	// Idempotency check: if order is not pending, do nothing.
	if order.Status != "pending" {
		tx.Rollback()
		// Reload to get the full object to return
		database.DB.Preload("User").Preload("Guest").Preload("Event").Preload("Payment").Preload("Items.Ticket.Event").First(&order, orderID)
		return &order, nil
	}

	// Update Payment and Order status
	if err := tx.Model(&models.Payment{}).Where("order_id = ?", orderID).Update("status", "paid").Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update payment status for free order: %w", err)
	}

	order.Status = "paid"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update order status for free order: %w", err)
	}

	// Fulfill the order (generate tickets, QR, PDF, and email)
	for i := range order.Items {
		if err := fulfillOrderItem(tx, &order, &order.Items[i]); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to fulfill free order item %d: %w", order.Items[i].ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload the order with all fulfilled data to return to the user
	if err := database.DB.Preload("User").Preload("Guest").Preload("Event").Preload("Payment").Preload("Items.Ticket.Event").First(&order, order.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload fulfilled order %d: %w", order.ID, err)
	}

	return &order, nil
}

// processPaidOrder is a private helper to centralize payment confirmation logic.
// It verifies the payment, updates the database, and fulfills the order.
// It is designed to be idempotent, meaning it can be called multiple times without side effects.
func processPaidOrder(reference string) (*models.Order, error) {
	// 1. Verify the transaction with BudPay
	verifyURL := fmt.Sprintf("%s/transaction/verify/%s", config.BudPayBaseURL, reference)
	req, _ := http.NewRequest("GET", verifyURL, nil)
	req.Header.Set("Authorization", "Bearer "+config.BudPaySecretKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify transaction with BudPay: %w", err)
	}

	defer resp.Body.Close()

	var verifyResp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&verifyResp)

	if !verifyResp.Status || verifyResp.Data.Status != "success" {
		return nil, fmt.Errorf("payment verification failed with BudPay: %s", verifyResp.Message)
	}

	// 2. Use a transaction to update database records and fulfill the order
	tx := database.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	var payment models.Payment
	if err := tx.Where("reference = ?", reference).First(&payment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("payment record not found for reference: %s", reference)
	}

	// Idempotency Check: If already paid, do nothing further but return the order.
	if payment.Status == "paid" {
		tx.Rollback() // No changes needed, so rollback.
		var existingOrder models.Order
		// Load the already-processed order to return it
		if err := database.DB.Preload("User").Preload("Guest").Preload("Event").Preload("Payment").Preload("Items.Ticket.Event").First(&existingOrder, payment.OrderID).Error; err != nil {
			return nil, fmt.Errorf("failed to load existing order %d: %w", payment.OrderID, err)
		}
		return &existingOrder, nil
	}

	// 3. Update Payment and Order status
	payment.Status = "paid"
	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	var order models.Order
	if err := tx.Preload("User").Preload("Guest").Preload("Event").Preload("Items.Ticket").First(&order, payment.OrderID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("order not found for payment: %d", payment.OrderID)
	}
	order.Status = "paid"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// 4. Fulfill the order (generate tickets, QR, PDF, and email)
	for i := range order.Items {
		if err := fulfillOrderItem(tx, &order, &order.Items[i]); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to fulfill order item %d: %w", order.Items[i].ID, err)
		}
	}

	// 5. Commit transaction
	return &order, tx.Commit().Error
}

// ✅ ConfirmPayment (Webhook)
func ConfirmPayment(c *gin.Context) {
	var webhook struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
		} `json:"data"`
	}

	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook"})
		return
	}

	if webhook.Event != "charge.success" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a payment success event"})
		return
	}

	// The webhook is the source of truth for fulfillment.
	// It will verify, update the DB, and send tickets.
	order, err := processPaidOrder(webhook.Data.Reference)
	if err != nil {
		// Log the internal error for monitoring
		fmt.Printf("Webhook processing error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during payment confirmation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Payment confirmed and tickets issued",
		"order_id": order.ID,
	})
}

// ✅ ConfirmPaymentGet (redirect after payment)
func ConfirmPaymentGet(c *gin.Context) {
	ref := c.Query("reference")

	if ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing reference"})
		return
	}

	// The redirect handler now just confirms the status for the user's UI.
	// It can also trigger fulfillment if the webhook somehow failed or was delayed.
	order, err := processPaidOrder(ref)
	if err != nil {
		fmt.Printf("Redirect verification error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while verifying your payment."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment verified and ticket(s) issued",
		"order": gin.H{
			"order_id":   order.ID,
			"reference":  ref,
			"status":     order.Status,
			"amount":     order.TotalAmount,
			"event_name": order.Event.Title,
			"event_date": order.Event.StartTime,
			"buyer_name": func() string {
				if order.User != nil {
					return order.User.Name
				} else if order.Guest != nil {
					return order.Guest.Name
				}
				return "Unknown"
			}(),
			"ticket_qty": len(order.Items),
			"tickets":    order.Items,
		},
	})
}

// fulfillOrderItem generates all assets for a ticket and sends the email.
// It operates within a database transaction.
func fulfillOrderItem(tx *gorm.DB, order *models.Order, item *models.OrderItem) error {
	// Explicitly load the ticket within the transaction to ensure we have the data for the update.
	var ticket models.Ticket
	if err := tx.First(&ticket, item.TicketID).Error; err != nil {
		return fmt.Errorf("could not find ticket with ID %d for order item: %w", item.TicketID, err)
	}

	code := uuid.New().String()
	relativeQrDir := filepath.Join("public", "qrcodes")
	absoluteQrDir := filepath.Join(utils.ProjectRoot, relativeQrDir)
	os.MkdirAll(absoluteQrDir, 0755)

	qrFilename := fmt.Sprintf("%s.png", code)
	relativeQrPath := filepath.Join(relativeQrDir, qrFilename)
	absoluteQrPath := filepath.Join(absoluteQrDir, qrFilename)

	if err := qrcode.WriteFile(code, qrcode.Medium, 256, absoluteQrPath); err != nil {
		return fmt.Errorf("failed to write QR code: %w", err)
	}

	relativePdfDir := filepath.Join("public", "tickets")
	absolutePdfDir := filepath.Join(utils.ProjectRoot, relativePdfDir)
	os.MkdirAll(absolutePdfDir, 0755)

	pdfFilename := fmt.Sprintf("ticket_%s.pdf", code)
	absolutePdfPath := filepath.Join(absolutePdfDir, pdfFilename)

	buyerName := "Guest"
	if order.User != nil {
		buyerName = order.User.Name
	} else if order.Guest != nil {
		buyerName = order.Guest.Name
	}
	eventDate := order.Event.StartTime.Format("January 2, 2006 3:04 PM")
	price := fmt.Sprintf("₦%.2f", item.UnitPrice)

	if err := utils.GenerateTicketPDF(order.Event.Title, buyerName, eventDate, order.Event.Venue, price, absoluteQrPath, absolutePdfPath); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	item.TicketCode = code
	item.QRCodePath = relativeQrPath
	if err := tx.Save(item).Error; err != nil {
		return fmt.Errorf("failed to save ticket code to order item: %w", err)
	}

	// ⭐️ Decrement the 'Quantity' on the parent Ticket to reflect the sale.
	// This is an atomic operation to prevent race conditions.
	if err := tx.Model(&ticket).Update("quantity", gorm.Expr("quantity - ?", item.Quantity)).Error; err != nil {
		return fmt.Errorf("failed to update ticket sold count: %w", err)
	}

	// Send email in a goroutine so it doesn't block the response.
	// The order object needs to be copied to avoid data races.
	orderCopy := *order
	itemCopy := *item
	go func() {
		utils.SendTicketEmail(orderCopy, itemCopy, absoluteQrPath, absolutePdfPath)
	}()

	return nil
}
