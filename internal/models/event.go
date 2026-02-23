package models

import "time"

type Event struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrganizerID uint      `gorm:"not null" json:"organizer_id"`
	CategoryID  uint      `json:"category_id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	EventImage  string    `gorm:"size:255" json:"event_image"`
	EventType   string    `gorm:"size:100" json:"event_type"`
	Location    string    `gorm:"size:255;not null" json:"location"`
	Description string    `gorm:"type:text" json:"description"`
	StartTime   time.Time `gorm:"not null" json:"start_time"`
	EndTime     time.Time `gorm:"not null" json:"end_time"`
	Venue       string    `gorm:"size:255" json:"venue"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	Category Category `gorm:"foreignKey:CategoryID" json:"category"`
}
