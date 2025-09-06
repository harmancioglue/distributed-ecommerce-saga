package domain

import (
	"fmt"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	"time"
)

type PaymentAggregate struct {
	*types.Payment
	SagaID          uuid.UUID  `json:"saga_id" db:"saga_id"`
	ExternalRef     string     `json:"external_ref,omitempty" db:"external_ref"` // Payment gateway reference
	FailureReason   string     `json:"failure_reason,omitempty" db:"failure_reason"`
	RefundedAmount  float64    `json:"refunded_amount" db:"refunded_amount"`
	RefundReference string     `json:"refund_reference,omitempty" db:"refund_reference"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	RefundedAt      *time.Time `json:"refunded_at,omitempty" db:"refunded_at"`
}

func NewPaymentAggregate(orderID, customerID, sagaID uuid.UUID, amount float64, paymentMethod string) *PaymentAggregate {
	return &PaymentAggregate{
		Payment: &types.Payment{
			ID:            uuid.New(),
			OrderID:       orderID,
			CustomerID:    customerID,
			Amount:        amount,
			PaymentMethod: paymentMethod,
			Status:        types.PaymentStatusPending,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		SagaID:         sagaID,
		RefundedAmount: 0.0,
	}
}

func (p *PaymentAggregate) ProcessPayment(transactionID, externalRef string) {
	p.Status = types.PaymentStatusCompleted
	p.TransactionID = transactionID
	p.ExternalRef = externalRef
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

func (p *PaymentAggregate) RefundPayment(refundRef string, amount float64) error {
	// Business validation
	if p.Status != types.PaymentStatusCompleted {
		return fmt.Errorf("only completed payments can be refunded, current status: %s", p.Status)
	}

	if amount <= 0 || amount > p.Amount {
		return fmt.Errorf("invalid refund amount: %.2f, max: %.2f", amount, p.Amount)
	}

	if p.RefundedAmount+amount > p.Amount {
		return fmt.Errorf("total refund amount limit exceed: %.2f + %.2f > %.2f",
			p.RefundedAmount, amount, p.Amount)
	}

	p.Status = types.PaymentStatusRefunded
	p.RefundedAmount += amount
	p.RefundReference = refundRef
	now := time.Now()
	p.RefundedAt = &now
	p.UpdatedAt = now

	return nil
}

func (p *PaymentAggregate) CanRefund() bool {
	return p.Status == types.PaymentStatusCompleted && p.RefundedAmount < p.Amount
}

func (p *PaymentAggregate) GetRemainingRefundAmount() float64 {
	return p.Amount - p.RefundedAmount
}

func (p *PaymentAggregate) IsFullyRefunded() bool {
	return p.RefundedAmount >= p.Amount
}

func (p *PaymentAggregate) FailPayment(reason string) {
	p.Status = types.PaymentStatusFailed
	p.FailureReason = reason
	p.UpdatedAt = time.Now()
}

type PaymentProcessRequest struct {
	SagaID        uuid.UUID `json:"saga_id"`
	OrderID       uuid.UUID `json:"order_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
}

// PaymentRefundRequest for saga
type PaymentRefundRequest struct {
	SagaID        uuid.UUID `json:"saga_id"`
	PaymentID     uuid.UUID `json:"payment_id,omitempty"`
	TransactionID string    `json:"transaction_id,omitempty"`
	Amount        float64   `json:"amount"`
	Reason        string    `json:"reason"`
}
