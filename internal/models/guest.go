package models

import "time"

type Guest struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Name      string    `gorm:"size:255;not null" json:"name"`
    Email     string    `gorm:"size:255;unique;not null" json:"email"`
    Phone     string    `gorm:"size:20" json:"phone"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
