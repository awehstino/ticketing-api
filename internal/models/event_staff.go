package models

import "time"

type EventStaff struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	EventID   uint      `gorm:"not null" json:"event_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Role      string    `gorm:"type:enum('staff');default:'staff'" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	User  User  `gorm:"foreignKey:UserID" json:"user"`
	Event Event `gorm:"foreignKey:EventID" json:"event"`
}

// TableName specifies the custom table name for the EventStaff model.
func (EventStaff) TableName() string {
	return "event_staff"
}
