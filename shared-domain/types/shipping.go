package types

import (
	"time"

	"github.com/google/uuid"
)

type ShippingStatus string

const (
	ShippingStatusPending   ShippingStatus = "pending"
	ShippingStatusPreparing ShippingStatus = "preparing"
	ShippingStatusShipped   ShippingStatus = "shipped"
	ShippingStatusDelivered ShippingStatus = "delivered"
	ShippingStatusCancelled ShippingStatus = "cancelled"
)

type Shipment struct {
	ID         uuid.UUID       `json:"id"`
	OrderID    uuid.UUID       `json:"order_id"`
	CustomerID uuid.UUID       `json:"customer_id"`
	Address    ShippingAddress `json:"address"`
	Status     ShippingStatus  `json:"status"`
	TrackingID string          `json:"tracking_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ShippingAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}
