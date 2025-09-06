package handlers

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/payment-service/internal/domain"
	"github.com/distributed-ecommerce-saga/payment-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	sharedHTTP "github.com/distributed-ecommerce-saga/shared-domain/http"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func (h *PaymentHandler) GetPaymentByOrderID(c *fiber.Ctx) error {
	orderIDStr := c.Params("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return sharedHTTP.BadRequestResponse(c, "Invalid order ID", map[string]interface{}{
			"order_id": orderIDStr,
		})
	}

	payment, err := h.paymentService.GetPaymentByOrderID(orderID)
	if err != nil {
		return sharedHTTP.NotFoundResponse(c, "Payment not found")
	}

	response := PaymentResponse{
		ID:              payment.ID,
		OrderID:         payment.OrderID,
		CustomerID:      payment.CustomerID,
		SagaID:          payment.SagaID,
		Amount:          payment.Amount,
		PaymentMethod:   payment.PaymentMethod,
		Status:          string(payment.Status),
		TransactionID:   payment.TransactionID,
		ExternalRef:     payment.ExternalRef,
		FailureReason:   payment.FailureReason,
		RefundedAmount:  payment.RefundedAmount,
		RefundReference: payment.RefundReference,
		CreatedAt:       payment.CreatedAt,
		UpdatedAt:       payment.UpdatedAt,
		ProcessedAt:     payment.ProcessedAt,
		RefundedAt:      payment.RefundedAt,
	}

	return sharedHTTP.SuccessResponse(c, "Payment retrieved successfully", response)
}

func (h *PaymentHandler) HealthCheck(c *fiber.Ctx) error {
	return sharedHTTP.SuccessResponse(c, "Payment service is healthy", map[string]interface{}{
		"service": "payment-service",
		"status":  "healthy",
	})
}

func (h *PaymentHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Payment service saga event received: %s from %s",
		event.EventType, event.Service)

	switch event.EventType {
	case "payment.process":
		return h.handlePaymentProcessCommand(event)

	case "payment.refund":
		return h.handlePaymentRefundCommand(event)

	default:
		log.Printf("Unhandled event type: %s", event.EventType)
		return nil
	}
}

func (h *PaymentHandler) handlePaymentProcessCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for payment.process", event)
	}

	request, err := h.mapToPaymentProcessRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Payload mapping error: %v", err), event)
	}

	if err := h.paymentService.ProcessPayment(request); err != nil {
		log.Printf("Payment processing error: %v", err)
		return err
	}

	return nil
}

func (h *PaymentHandler) handlePaymentRefundCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for payment.refund", event)
	}

	request, err := h.mapToPaymentRefundRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Refund payload mapping error: %v", err), event)
	}

	if err := h.paymentService.ProcessRefund(request); err != nil {
		log.Printf("Payment refund error: %v", err)
		return err
	}

	return nil
}

func (h *PaymentHandler) mapToPaymentProcessRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.PaymentProcessRequest, error) {
	request := domain.PaymentProcessRequest{
		SagaID: sagaID,
	}

	if orderIDStr, ok := payload["order_id"].(string); ok {
		if orderID, err := uuid.Parse(orderIDStr); err == nil {
			request.OrderID = orderID
		} else {
			return request, fmt.Errorf("invalid order_id format: %s", orderIDStr)
		}
	} else {
		return request, fmt.Errorf("missing or invalid order_id")
	}

	if customerIDStr, ok := payload["customer_id"].(string); ok {
		if customerID, err := uuid.Parse(customerIDStr); err == nil {
			request.CustomerID = customerID
		} else {
			return request, fmt.Errorf("invalid customer_id format: %s", customerIDStr)
		}
	} else {
		return request, fmt.Errorf("missing or invalid customer_id")
	}

	if amount, ok := payload["amount"].(float64); ok {
		request.Amount = amount
	} else {
		return request, fmt.Errorf("missing or invalid amount")
	}

	if method, ok := payload["payment_method"].(string); ok {
		request.PaymentMethod = method
	} else {
		request.PaymentMethod = "credit_card" // Default
	}

	return request, nil
}

func (h *PaymentHandler) mapToPaymentRefundRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.PaymentRefundRequest, error) {
	request := domain.PaymentRefundRequest{
		SagaID: sagaID,
	}

	if amount, ok := payload["amount"].(float64); ok {
		request.Amount = amount
	} else {
		return request, fmt.Errorf("missing or invalid refund amount")
	}

	if reason, ok := payload["reason"].(string); ok {
		request.Reason = reason
	}

	if txnID, ok := payload["transaction_id"].(string); ok {
		request.TransactionID = txnID
	}

	if paymentIDStr, ok := payload["payment_id"].(string); ok {
		if paymentID, err := uuid.Parse(paymentIDStr); err == nil {
			request.PaymentID = paymentID
		}
	}

	return request, nil
}

func (h *PaymentHandler) logAndReturnError(message string, event events.SagaEvent) error {
	log.Printf("%s - Event: %+v", message, event)
	return fmt.Errorf(message)
}

func (h *PaymentHandler) StartConsuming(consumer *messaging.Consumer) error {
	routingKeys := []string{
		"saga.saga-orchestrator.payment.process", // Payment process command
		"saga.saga-orchestrator.payment.refund",  // Refund command
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}
