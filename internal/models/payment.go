package models

import "time"

type Payment struct {
    ID        uint       `gorm:"primaryKey" json:"id"`
    OrderID   uint       `json:"order_id"`
    UserID    *uint      `json:"user_id"`
    GuestID   *uint      `json:"guest_id"`
    EventID   uint       `json:"event_id"`
    TicketID  uint       `json:"ticket_id"`
    Amount    float64    `json:"amount"`
    Currency  string     `gorm:"default:'NGN'" json:"currency"`
    Status    string     `gorm:"type:enum('pending','paid','failed');default:'pending'" json:"status"`
    Reference string     `gorm:"unique" json:"reference"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

