package types

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusFailed     OrderStatus = "failed"
)

type Order struct {
	ID              uuid.UUID        `json:"id"`
	CustomerID      uuid.UUID        `json:"customer_id"`
	Items           []OrderItem      `json:"items"`
	TotalAmount     float64          `json:"total_amount"`
	Status          OrderStatus      `json:"status"`
	ShippingAddress *ShippingAddress `json:"shipping_address"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type OrderItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
}
