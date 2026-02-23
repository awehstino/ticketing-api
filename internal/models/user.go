package models

import "time"

type User struct {
    ID                uint      `gorm:"primaryKey"`
    Name              string    `gorm:"size:255;not null"`
    Email             string    `gorm:"size:255;unique;not null"`
    PasswordHash      string    `gorm:"size:255;not null"`
    Role              string    `gorm:"type:enum('attendee','organizer','admin');default:'attendee'"`
    AccountType       string    `gorm:"type:enum('personal','business');default:'personal'"`
    VerificationToken string    `gorm:"size:255"`
    IsVerified        bool      `gorm:"default:false"`
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
