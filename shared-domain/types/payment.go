package types

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID            uuid.UUID     `json:"id"`
	OrderID       uuid.UUID     `json:"order_id"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	Amount        float64       `json:"amount"`
	PaymentMethod string        `json:"payment_method"`
	Status        PaymentStatus `json:"status"`
	TransactionID string        `json:"transaction_id,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
