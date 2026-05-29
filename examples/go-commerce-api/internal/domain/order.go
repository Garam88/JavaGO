package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
)

type Item struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Stock      int       `json:"stock"`
	PriceCents int       `json:"price_cents"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	ItemID    string      `json:"item_id"`
	Quantity  int         `json:"quantity"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderCreatedEvent struct {
	EventID   string    `json:"event_id"`
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ItemID    string    `json:"item_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusPublishing OutboxStatus = "publishing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
)

const OutboxTopicOrderCreated = "order.created"

type OutboxEvent struct {
	ID          string       `json:"id"`
	Topic       string       `json:"topic"`
	Payload     []byte       `json:"payload"`
	Status      OutboxStatus `json:"status"`
	Attempts    int          `json:"attempts"`
	LastError   string       `json:"last_error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	PublishedAt *time.Time   `json:"published_at,omitempty"`
}
