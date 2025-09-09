package service

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/payment-service/internal/domain"
	"github.com/distributed-ecommerce-saga/payment-service/internal/gateway"
	"github.com/distributed-ecommerce-saga/payment-service/internal/repository"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/google/uuid"
)

type PaymentService struct {
	paymentRepo    *repository.PaymentRepository
	paymentGateway gateway.PaymentGateway
	publisher      *messaging.Publisher
}

func NewPaymentService(
	paymentRepo *repository.PaymentRepository,
	paymentGateway gateway.PaymentGateway,
	publisher *messaging.Publisher,
) *PaymentService {
	return &PaymentService{
		paymentRepo:    paymentRepo,
		paymentGateway: paymentGateway,
		publisher:      publisher,
	}
}

// ProcessPayment Process payment.process command which receives from saga
func (s *PaymentService) ProcessPayment(request domain.PaymentProcessRequest) error {
	log.Printf("Payment process started: OrderID=%s, Amount=%.2f",
		request.OrderID, request.Amount)

	// Business validation
	if request.Amount <= 0 {
		return s.publishPaymentFailedEvent(request.SagaID, request.OrderID,
			"Invalid payment amount", request.Amount)
	}

	payment := domain.NewPaymentAggregate(
		request.OrderID,
		request.CustomerID,
		request.SagaID,
		request.Amount,
		request.PaymentMethod,
	)

	if err := s.paymentRepo.CreatePayment(payment); err != nil {
		return s.publishPaymentFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Database error: %v", err), request.Amount)
	}

	gatewayRequest := gateway.PaymentRequest{
		OrderID:       request.OrderID,
		CustomerID:    request.CustomerID,
		Amount:        request.Amount,
		Currency:      "USD", // Default currency
		PaymentMethod: request.PaymentMethod,
		Description:   fmt.Sprintf("Order payment for %s", request.OrderID),
	}

	gatewayResponse, err := s.paymentGateway.ProcessPayment(gatewayRequest)
	if err != nil {
		// Gateway error - payment'i failed olarak işaretle
		payment.FailPayment(fmt.Sprintf("Gateway error: %v", err))
		s.paymentRepo.UpdatePayment(payment)

		return s.publishPaymentFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Payment gateway error: %v", err), request.Amount)
	}

	// Gateway response'una göre işle
	if !gatewayResponse.Success {
		// Payment failed
		payment.FailPayment(gatewayResponse.FailureReason)
		s.paymentRepo.UpdatePayment(payment)

		return s.publishPaymentFailedEvent(request.SagaID, request.OrderID,
			gatewayResponse.FailureReason, request.Amount)
	}

	// Payment successful
	payment.ProcessPayment(gatewayResponse.TransactionID, gatewayResponse.ExternalRef)
	if err := s.paymentRepo.UpdatePayment(payment); err != nil {
		log.Printf("Payment success update error: %v", err)
		// Burada compensating action yapılabilir
	}

	// Success event publish et
	return s.publishPaymentProcessedEvent(payment)
}

// ProcessRefund Process payment.refund command which receives from saga
func (s *PaymentService) ProcessRefund(request domain.PaymentRefundRequest) error {
	log.Printf("Payment refund başlatıldı: SagaID=%s, Amount=%.2f",
		request.SagaID, request.Amount)

	var payment *domain.PaymentAggregate
	var err error

	if request.PaymentID != uuid.Nil {
		payment, err = s.paymentRepo.GetPaymentByID(request.PaymentID)
	} else if request.TransactionID != "" {
		// Transaction ID ile arama için ek metod gerekli, şimdilik saga ID kullan
		payments, err := s.paymentRepo.GetPaymentsBySagaID(request.SagaID)
		if err == nil && len(payments) > 0 {
			payment = payments[0] // İlk completed payment'i al
		}
	}

	if err != nil {
		return s.publishRefundFailedEvent(request.SagaID,
			fmt.Sprintf("Payment bulunamadı: %v", err))
	}

	if !payment.CanRefund() {
		return s.publishRefundFailedEvent(request.SagaID,
			fmt.Sprintf("Payment refund edilemez, status: %s", payment.Status))
	}

	refundAmount := request.Amount
	if refundAmount <= 0 || refundAmount > payment.GetRemainingRefundAmount() {
		return s.publishRefundFailedEvent(request.SagaID,
			fmt.Sprintf("Geçersiz refund amount: %.2f", refundAmount))
	}

	gatewayRequest := gateway.RefundRequest{
		OriginalTransactionID: payment.TransactionID,
		ExternalRef:           payment.ExternalRef,
		Amount:                refundAmount,
		Reason:                request.Reason,
	}

	gatewayResponse, err := s.paymentGateway.RefundPayment(gatewayRequest)
	if err != nil {
		return s.publishRefundFailedEvent(request.SagaID,
			fmt.Sprintf("Gateway refund error: %v", err))
	}

	if !gatewayResponse.Success {
		return s.publishRefundFailedEvent(request.SagaID,
			gatewayResponse.FailureReason)
	}

	if err := payment.RefundPayment(gatewayResponse.RefundReference, refundAmount); err != nil {
		return s.publishRefundFailedEvent(request.SagaID,
			fmt.Sprintf("Refund processing error: %v", err))
	}

	if err := s.paymentRepo.UpdatePayment(payment); err != nil {
		log.Printf("Refund database update hatası: %v", err)
	}

	return s.publishPaymentRefundedEvent(payment, refundAmount)
}

func (s *PaymentService) GetPaymentByOrderID(orderID uuid.UUID) (*domain.PaymentAggregate, error) {
	return s.paymentRepo.GetPaymentByOrderID(orderID)
}

// publishPaymentProcessedEvent publish event of successfully payment

func (s *PaymentService) publishPaymentProcessedEvent(payment *domain.PaymentAggregate) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        payment.SagaID,
		OrderID:       payment.OrderID,
		EventType:     events.PaymentProcessedEvent,
		Service:       "payment-service",
		CorrelationID: uuid.New(),
		Payload: events.PaymentProcessedPayload{
			Payment: *payment.Payment,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("payment processed event publish error: %v", err)
	}

	log.Printf("Payment processed event published: PaymentID=%s, OrderID=%s",
		payment.ID, payment.OrderID)
	return nil
}

func (s *PaymentService) publishPaymentFailedEvent(sagaID, orderID uuid.UUID, reason string, amount float64) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.PaymentFailedEvent,
		Service:       "payment-service",
		CorrelationID: uuid.New(),
		Payload: events.PaymentFailedPayload{
			OrderID: orderID,
			Reason:  reason,
			Amount:  amount,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("payment failed event publish error: %v", err)
	}

	log.Printf("Payment failed event published: OrderID=%s, Reason=%s",
		orderID, reason)
	return nil
}

func (s *PaymentService) publishPaymentRefundedEvent(payment *domain.PaymentAggregate, refundAmount float64) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        payment.SagaID,
		OrderID:       payment.OrderID,
		EventType:     events.PaymentRefundedEvent,
		Service:       "payment-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"payment_id":       payment.ID,
			"transaction_id":   payment.TransactionID,
			"refund_reference": payment.RefundReference,
			"refunded_amount":  refundAmount,
			"total_refunded":   payment.RefundedAmount,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("payment refunded event publish error: %v", err)
	}

	log.Printf("Payment refunded event published: PaymentID=%s, Amount=%.2f",
		payment.ID, refundAmount)
	return nil
}

func (s *PaymentService) publishRefundFailedEvent(sagaID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       uuid.Nil, // OrderID moy not known
		EventType:     "payment.refund.failed",
		Service:       "payment-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"reason": reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("refund failed event publish error: %v", err)
	}

	log.Printf("Refund failed event published: SagaID=%s, Reason=%s",
		sagaID, reason)
	return nil
}
