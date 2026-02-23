package models

import "time"

type Ticket struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     uint      `json:"event_id"`
	Event       Event     `gorm:"foreignKey:EventID" json:"event"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `gorm:"type:varchar(20);default:'paid'" json:"type"` // 'free' or 'paid'
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Sold        int       `json:"sold"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
