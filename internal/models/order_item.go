package models

import "time"

type OrderItem struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	OrderID     uint       `json:"order_id"`
	TicketID    uint       `json:"ticket_id"`
	Quantity    int        `json:"quantity"`
	UnitPrice   float64    `json:"unit_price"`
	TotalPrice  float64    `json:"total_price"`
	TicketCode  string     `json:"ticket_code"`
	QRCodePath  string     `json:"qr_code_path"`
	CheckedIn   bool       `gorm:"default:false" json:"checked_in"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	CheckedInAt *time.Time `json:"checked_in_at"`

	// ✅ Relations
	Ticket *Ticket `gorm:"foreignKey:TicketID" json:"ticket"`
	Order  *Order  `gorm:"foreignKey:OrderID" json:"order"`
}
