package gateway

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// PaymentGateway external payment provider interface
type PaymentGateway interface {
	ProcessPayment(request PaymentRequest) (*PaymentResponse, error)
	RefundPayment(request RefundRequest) (*RefundResponse, error)
	GetPaymentStatus(externalRef string) (*PaymentStatusResponse, error)
}

type PaymentRequest struct {
	OrderID       uuid.UUID `json:"order_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PaymentMethod string    `json:"payment_method"`
	Description   string    `json:"description"`
}

type PaymentResponse struct {
	Success       bool      `json:"success"`
	TransactionID string    `json:"transaction_id"`
	ExternalRef   string    `json:"external_ref"`
	Status        string    `json:"status"`
	Amount        float64   `json:"amount"`
	ProcessedAt   time.Time `json:"processed_at"`
	FailureReason string    `json:"failure_reason,omitempty"`
}

type RefundRequest struct {
	OriginalTransactionID string  `json:"original_transaction_id"`
	ExternalRef           string  `json:"external_ref"`
	Amount                float64 `json:"amount"`
	Reason                string  `json:"reason"`
}

type RefundResponse struct {
	Success         bool      `json:"success"`
	RefundID        string    `json:"refund_id"`
	RefundReference string    `json:"refund_reference"`
	Amount          float64   `json:"amount"`
	RefundedAt      time.Time `json:"refunded_at"`
	FailureReason   string    `json:"failure_reason,omitempty"`
}

type PaymentStatusResponse struct {
	Status        string    `json:"status"`
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	ProcessedAt   time.Time `json:"processed_at"`
}

// MockPaymentGateway mock payment gateway for test
type MockPaymentGateway struct {
	FailureRate float64 // 0.0 - 1.0 arası hata oranı
}

func NewMockPaymentGateway(failureRate float64) *MockPaymentGateway {
	return &MockPaymentGateway{
		FailureRate: failureRate,
	}
}

func (m *MockPaymentGateway) ProcessPayment(request PaymentRequest) (*PaymentResponse, error) {
	log.Printf("Mock Payment Gateway: Processing payment for Order %s, Amount: %.2f",
		request.OrderID, request.Amount)

	// Simulate processing delay
	time.Sleep(time.Millisecond * 500)

	// Random failure simulation
	if rand.Float64() < m.FailureRate {
		return &PaymentResponse{
			Success:       false,
			Status:        "failed",
			Amount:        request.Amount,
			ProcessedAt:   time.Now(),
			FailureReason: "Insufficient funds", // Mock failure reason
		}, nil
	}

	transactionID := fmt.Sprintf("TXN_%d", time.Now().Unix())
	externalRef := fmt.Sprintf("REF_%s", uuid.New().String()[:8])

	return &PaymentResponse{
		Success:       true,
		TransactionID: transactionID,
		ExternalRef:   externalRef,
		Status:        "completed",
		Amount:        request.Amount,
		ProcessedAt:   time.Now(),
	}, nil
}

func (m *MockPaymentGateway) RefundPayment(request RefundRequest) (*RefundResponse, error) {
	log.Printf("Mock Payment Gateway: Processing refund for Transaction %s, Amount: %.2f",
		request.OriginalTransactionID, request.Amount)

	time.Sleep(time.Millisecond * 300)

	if rand.Float64() < (m.FailureRate * 0.5) {
		return &RefundResponse{
			Success:       false,
			Amount:        request.Amount,
			RefundedAt:    time.Now(),
			FailureReason: "Refund not allowed for this transaction",
		}, nil
	}

	refundID := fmt.Sprintf("RFD_%d", time.Now().Unix())
	refundRef := fmt.Sprintf("RREF_%s", uuid.New().String()[:8])

	return &RefundResponse{
		Success:         true,
		RefundID:        refundID,
		RefundReference: refundRef,
		Amount:          request.Amount,
		RefundedAt:      time.Now(),
	}, nil
}

func (m *MockPaymentGateway) GetPaymentStatus(externalRef string) (*PaymentStatusResponse, error) {
	log.Printf("Mock Payment Gateway: Checking status for Reference %s", externalRef)

	time.Sleep(time.Millisecond * 200)

	return &PaymentStatusResponse{
		Status:        "completed",
		TransactionID: fmt.Sprintf("TXN_%s", externalRef),
		Amount:        100.00, // Mock amount
		ProcessedAt:   time.Now(),
	}, nil
}
