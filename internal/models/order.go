package models

import "time"

type Order struct {
    ID          uint        `gorm:"primaryKey" json:"id"`
    UserID      *uint       `json:"user_id"`
    GuestID     *uint       `json:"guest_id"`
    EventID     uint        `gorm:"not null" json:"event_id"`
    TotalAmount float64     `gorm:"not null" json:"total_amount"`
    Status      string      `gorm:"type:enum('pending','paid','cancelled');default:'pending'" json:"status"`
    CreatedAt   time.Time   `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time   `gorm:"autoUpdateTime" json:"updated_at"`

    // Relations
    User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Guest  *Guest  `gorm:"foreignKey:GuestID" json:"guest,omitempty"`
    Event  Event   `gorm:"foreignKey:EventID" json:"event"`

    


	Items []OrderItem `json:"items"`
    Payment Payment `json:"payment"`
}
