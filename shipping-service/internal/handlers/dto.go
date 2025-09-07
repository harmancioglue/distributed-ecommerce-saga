package handlers

import (
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type ShipmentResponse struct {
	ID            uuid.UUID             `json:"id"`
	OrderID       uuid.UUID             `json:"order_id"`
	CustomerID    uuid.UUID             `json:"customer_id"`
	SagaID        uuid.UUID             `json:"saga_id"`
	Status        string                `json:"status"`
	TrackingID    string                `json:"tracking_id"`
	Address       types.ShippingAddress `json:"address"`
	FailureReason string                `json:"failure_reason,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}
