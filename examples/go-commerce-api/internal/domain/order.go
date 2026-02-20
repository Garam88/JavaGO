package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
)

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	ItemID    string      `json:"item_id"`
	Quantity  int         `json:"quantity"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderCreatedEvent struct {
	OrderID   string
	UserID    string
	ItemID    string
	Quantity  int
	CreatedAt time.Time
}
