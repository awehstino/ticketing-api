package controllers

import (
	"fmt"
	"net/http"

	"github.com/awehstino/ticketing-api/internal/config"
	"github.com/awehstino/ticketing-api/internal/database"
	"github.com/awehstino/ticketing-api/internal/models"
	"github.com/awehstino/ticketing-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegisterInput defines the expected fields for user registration
type RegisterInput struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required"`
	AccountType string `json:"account_type" binding:"required"`
}

// Register new user (with auto-login)
func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// Generate verification token
	verificationToken := uuid.New().String()

	// Determine role based on account type
	role := "attendee"
	if input.AccountType == "business" {
		role = "organizer"
	}

	user := models.User{
		Name:              input.Name,
		Email:             input.Email,
		PasswordHash:      string(hashedPassword),
		Role:              role,
		AccountType:       input.AccountType,
		VerificationToken: verificationToken,
		IsVerified:        false,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send verification email
	verificationLink := fmt.Sprintf("%s/verify?token=%s", config.AppBaseURL, verificationToken)
	go utils.SendEmail(user.Email, "Confirm your email", "Click this link to verify your email: "+verificationLink)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Account created! Please check your email to verify your account.",
	})
}

// LoginInput defines the expected fields for user login
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// JWT Claims
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Login user
func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user is verified
	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Please verify your email before logging in"})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"role":         user.Role,
			"account_type": user.AccountType,
		},
	})
}

// ScannerLoginInput defines the expected fields for the passwordless scanner login
type ScannerLoginInput struct {
	Email string `json:"email" binding:"required,email"`
}

// ScannerLogin handles passwordless authentication for event staff.
// It verifies a user exists and is assigned as staff to at least one event.
func ScannerLogin(c *gin.Context) {
	var input ScannerLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Find user by email
	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access Denied: User not found."})
		return
	}

	// 2. Verify the user is a staff member for at least one event
	var staffRecord models.EventStaff
	if err := database.DB.Where("user_id = ?", user.ID).First(&staffRecord).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: This user is not registered as event staff."})
		return
	}

	// 3. If checks pass, generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Staff login successful",
		"token":   token,
		"user": gin.H{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"role":         user.Role,
			"account_type": user.AccountType,
		},
	})
}

// Profile handler - fetch user details from DB using claims
func Profile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"account_type": user.AccountType,
		"role":         user.Role,
	})
}

func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
		return
	}

	var user models.User
	if err := database.DB.Where("verification_token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid token"})
		return
	}

	user.IsVerified = true
	user.VerificationToken = ""
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully!"})
}
