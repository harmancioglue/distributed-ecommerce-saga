package handlers

import (
	"time"

	"github.com/google/uuid"
)

type PaymentResponse struct {
	ID              uuid.UUID  `json:"id"`
	OrderID         uuid.UUID  `json:"order_id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	SagaID          uuid.UUID  `json:"saga_id"`
	Amount          float64    `json:"amount"`
	PaymentMethod   string     `json:"payment_method"`
	Status          string     `json:"status"`
	TransactionID   string     `json:"transaction_id,omitempty"`
	ExternalRef     string     `json:"external_ref,omitempty"`
	FailureReason   string     `json:"failure_reason,omitempty"`
	RefundedAmount  float64    `json:"refunded_amount"`
	RefundReference string     `json:"refund_reference,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	RefundedAt      *time.Time `json:"refunded_at,omitempty"`
}

type PaymentStatusResponse struct {
	Status          string    `json:"status"`
	CanRefund       bool      `json:"can_refund"`
	RemainingRefund float64   `json:"remaining_refund_amount"`
	IsFullyRefunded bool      `json:"is_fully_refunded"`
	LastUpdated     time.Time `json:"last_updated"`
}

type RefundRequest struct {
	Amount float64 `json:"amount" validate:"required,min=0.01"`
	Reason string  `json:"reason" validate:"required,min=3"`
}

type RefundResponse struct {
	RefundID        string    `json:"refund_id"`
	RefundReference string    `json:"refund_reference"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	RefundedAt      time.Time `json:"refunded_at"`
}
