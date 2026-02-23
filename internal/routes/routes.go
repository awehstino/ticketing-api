package routes

import (
	"time"

	"github.com/awehstino/ticketing-api/internal/handlers/controllers"
	"github.com/awehstino/ticketing-api/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Apply a general rate limiter to all requests (e.g., 100 requests per minute per IP)
	r.Use(middleware.RateLimiter(100, time.Minute))

	// Public routes
	authLimiter := middleware.RateLimiter(5, time.Minute) // Stricter: 5 requests per minute
	r.POST("/register", authLimiter, controllers.Register)
	r.POST("/login", authLimiter, controllers.Login)
	r.POST("/scanner-login", authLimiter, controllers.ScannerLogin) // New passwordless login for staff
	r.GET("/verify-email", controllers.VerifyEmail)
	r.GET("/verify", controllers.VerifyEmail)

	// Public Event & Ticket routes
	r.GET("/events", controllers.ListEvents)
	r.GET("/events/:event_id", controllers.GetEvent)
	r.GET("/events/:event_id/tickets", controllers.ListTickets)
	r.GET("/tickets/:ticket_id", controllers.GetTicket)
	// Public Category routes
	r.GET("/categories", controllers.ListCategories)
	r.GET("/categories/:id", controllers.GetCategory)

	// Checkout & payment
	r.POST("/checkout", middleware.OptionalAuthMiddleware(), controllers.Checkout) // create order + initiate payment
	r.POST("/payments/confirm", controllers.ConfirmPayment)                        // webhook / manual confirm
	r.GET("/payments/confirm", controllers.ConfirmPaymentGet)

	// Serve static files (like scanner.html)
	r.Static("/public", "./public")
	r.GET("/scanner", func(c *gin.Context) {
		c.File("./public/scanner.html")
	})

	// Protected routes
	auth := r.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/profile", controllers.Profile)

		// Events & tickets (organizer/admin only)
		auth.POST("/events", middleware.RequireRole("organizer", "admin"), controllers.CreateEvent)
		auth.PUT("/events/:event_id", middleware.RequireRole("organizer", "admin"), controllers.UpdateEvent)
		auth.DELETE("/events/:event_id", middleware.RequireRole("organizer", "admin"), controllers.DeleteEvent)
		auth.POST("/events/:event_id/tickets", controllers.CreateTicket)
		auth.PUT("/tickets/:ticket_id", middleware.RequireRole("organizer", "admin"), controllers.UpdateTicket)
		auth.DELETE("/tickets/:ticket_id", middleware.RequireRole("organizer", "admin"), controllers.DeleteTicket)

		// Attendees & check-in
		auth.GET("/events/:event_id/attendees", middleware.RequireRole("organizer", "admin"), controllers.GetAttendeesReport)
		auth.POST("/events/:event_id/checkin", middleware.RequireEventStaffOrRole("organizer", "admin", "staff"), controllers.CheckInTicket)
		auth.DELETE("/events/:event_id/checkin/:ticket_code", middleware.RequireEventStaffOrRole("organizer", "admin", "staff"), controllers.RevokeCheckIn)

		// Event staff management
		auth.POST("/events/:event_id/staff", middleware.RequireRole("organizer", "admin"), controllers.AddStaffToEvent)
		auth.GET("/events/:event_id/staff", middleware.RequireRole("organizer", "admin"), controllers.ListEventStaff)
		auth.GET("/scanner/events", controllers.GetScannerEvents) // Endpoint for scanner to fetch user-specific events
		auth.DELETE("/events/:event_id/staff/:userID", middleware.RequireRole("organizer", "admin"), controllers.RemoveStaffFromEvent)

		// Category Management (admin/organizer only)
		auth.POST("/categories", middleware.RequireRole("organizer", "admin"), controllers.CreateCategory)
		auth.PUT("/categories/:id", middleware.RequireRole("organizer", "admin"), controllers.UpdateCategory)
		auth.DELETE("/categories/:id", middleware.RequireRole("organizer", "admin"), controllers.DeleteCategory)
	}
}
