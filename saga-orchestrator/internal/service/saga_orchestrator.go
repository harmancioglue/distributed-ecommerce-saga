package service

import (
	"fmt"
	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/domain"
	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/repository"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	"log"
	"time"
)

type SagaOrchestrator struct {
	sagaRepo  *repository.SagaRepository
	publisher *messaging.Publisher
}

func NewSagaOrchestrator(sagaRepo *repository.SagaRepository, publisher *messaging.Publisher) *SagaOrchestrator {
	return &SagaOrchestrator{
		sagaRepo:  sagaRepo,
		publisher: publisher,
	}
}

func (s *SagaOrchestrator) StartSaga(order types.Order) error {
	sagaID := uuid.New()

	saga := &domain.SagaInstance{
		ID:             sagaID,
		OrderID:        order.ID,
		CustomerID:     order.CustomerID,
		Status:         domain.SagaStatusStarted,
		CurrentStep:    domain.StepOrderCreated,
		CompletedSteps: []domain.SagaStep{domain.StepOrderCreated},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Context: map[string]interface{}{
			"order":        order,
			"total_amount": order.TotalAmount,
			"items":        order.Items,
			"retry_counts": map[string]int{},
		},
	}

	if err := s.sagaRepo.CreateSaga(saga); err != nil {
		return err
	}

	log.Printf("Saga started: SagaID=%s, OrderID=%s", sagaID, order.ID)

	return s.processNextStep(saga)
}

func (s *SagaOrchestrator) processNextStep(saga *domain.SagaInstance) error {
	nextStep := saga.GetNextStep()
	if nextStep == "" {
		return s.completeSaga(saga)
	}

	saga.Status = domain.SagaStatusInProgress
	saga.UpdatedAt = time.Now()

	if err := s.sagaRepo.UpdateSaga(saga); err != nil {
		return err
	}

	return s.sendStepEvent(saga, nextStep)

}

func (s *SagaOrchestrator) HandleCompensationSuccess(sagaID uuid.UUID, completedCompensation domain.SagaStep) error {
	saga, err := s.sagaRepo.GetSagaByID(sagaID)
	if err != nil {
		return err
	}

	saga.MarkCompensationCompleted(completedCompensation)

	log.Printf("✅ Compensation step completed: %s", completedCompensation)

	return s.startCompensation(saga)
}

func (s *SagaOrchestrator) completeSaga(saga *domain.SagaInstance) error {
	saga.Status = domain.SagaStatusCompleted
	saga.UpdatedAt = time.Now()
	now := time.Now()
	saga.CompletedAt = &now

	if err := s.sagaRepo.UpdateSaga(saga); err != nil {
		return fmt.Errorf("saga tamamlama hatası: %v", err)
	}

	log.Printf("Saga completed successfully: SagaID=%s, OrderID=%s", saga.ID, saga.OrderID)

	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        saga.ID,
		OrderID:       saga.OrderID,
		EventType:     events.OrderCompletedEvent,
		Service:       "saga-orchestrator",
		Timestamp:     time.Now(),
		CorrelationID: uuid.New(),
		Payload: events.OrderCompletedPayload{
			OrderID: saga.OrderID,
			Status:  "completed",
		},
	}

	return s.publisher.PublishSagaEvent(event)
}

func (s *SagaOrchestrator) ProcessIncomingEvent(event events.SagaEvent) error {
	log.Printf("Event received: %s from %s", event.EventType, event.Service)

	// Event type'a göre işle
	switch event.EventType {

	// Success events
	case events.PaymentProcessedEvent:
		return s.HandleStepSuccess(event.SagaID, domain.StepPaymentProcessed,
			event.Payload.(map[string]interface{}))

	case events.InventoryReservedEvent:
		return s.HandleStepSuccess(event.SagaID, domain.StepInventoryReserved,
			event.Payload.(map[string]interface{}))

	case events.ShippingCreatedEvent:
		return s.HandleStepSuccess(event.SagaID, domain.StepShippingCreated,
			event.Payload.(map[string]interface{}))

	case events.NotificationSentEvent:
		return s.HandleStepSuccess(event.SagaID, domain.StepNotificationSent,
			event.Payload.(map[string]interface{}))

	// Failure events
	case events.PaymentFailedEvent:
		return s.HandleStepFailure(event.SagaID, domain.StepPaymentProcessed,
			event.Payload.(map[string]interface{}))

	case events.InventoryFailedEvent:
		return s.HandleStepFailure(event.SagaID, domain.StepInventoryReserved,
			event.Payload.(map[string]interface{}))

	// COMPENSATION SUCCESS EVENTS
	case events.ShippingCancelledEvent:
		return s.HandleCompensationSuccess(event.SagaID, domain.StepShippingCancelled)

	case events.InventoryReleasedEvent:
		return s.HandleCompensationSuccess(event.SagaID, domain.StepInventoryReleased)

	case events.PaymentRefundedEvent:
		return s.HandleCompensationSuccess(event.SagaID, domain.StepPaymentRefunded)

	default:
		log.Printf("Unknown event type: %s", event.EventType)
		return nil
	}
}

func (s *SagaOrchestrator) HandleStepFailure(sagaID uuid.UUID, failedStep domain.SagaStep, eventData map[string]interface{}) error {
	saga, err := s.sagaRepo.GetSagaByID(sagaID)
	if err != nil {
		return fmt.Errorf("saga not found: %v", err)
	}

	// Failure reason'u kaydet
	if reason, ok := eventData["reason"].(string); ok {
		saga.FailureReason = reason
	}

	saga.Status = domain.SagaStatusCompensating
	saga.UpdatedAt = time.Now()

	log.Printf("Step failure: %s for SagaID=%s, compensation could not start", failedStep, sagaID)

	return s.startCompensation(saga)
}

func (s *SagaOrchestrator) HandleStepSuccess(sagaID uuid.UUID, completedStep domain.SagaStep, eventData map[string]interface{}) error {
	saga, err := s.sagaRepo.GetSagaByID(sagaID)
	if err != nil {
		return fmt.Errorf("saga not found: %v", err)
	}

	saga.MarkStepCompleted(completedStep)

	if eventData != nil {
		switch completedStep {
		case domain.StepPaymentProcessed:
			saga.Context["payment_id"] = eventData["payment_id"]
			saga.Context["transaction_id"] = eventData["transaction_id"]
		case domain.StepInventoryReserved:
			saga.Context["reservation_ids"] = eventData["reservation_ids"]
		case domain.StepShippingCreated:
			saga.Context["shipment_id"] = eventData["shipment_id"]
			saga.Context["tracking_id"] = eventData["tracking_id"]
		}
	}

	log.Printf("Step completed: %s for SagaID=%s", completedStep, sagaID)

	return s.processNextStep(saga)
}

func (s *SagaOrchestrator) startCompensation(saga *domain.SagaInstance) error {
	compensationStep := saga.GetCompensationStep()
	if compensationStep == "" {
		return s.compensationCompleted(saga)
	}

	log.Printf("Compensation started: %s for SagaID=%s", compensationStep, saga.ID)

	if err := s.sagaRepo.UpdateSaga(saga); err != nil {
		return fmt.Errorf("saga compensation update error: %v", err)
	}

	return s.sendCompensationEvent(saga, compensationStep)
}

func (s *SagaOrchestrator) compensationCompleted(saga *domain.SagaInstance) error {
	saga.Status = domain.SagaStatusCompensated
	saga.UpdatedAt = time.Now()
	now := time.Now()
	saga.CompletedAt = &now

	if err := s.sagaRepo.UpdateSaga(saga); err != nil {
		return fmt.Errorf("saga compensation complete error: %v", err)
	}

	log.Printf("Saga compensation completed: SagaID=%s, OrderID=%s", saga.ID, saga.OrderID)

	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        saga.ID,
		OrderID:       saga.OrderID,
		EventType:     events.OrderCancelledEvent,
		Service:       "saga-orchestrator",
		Timestamp:     time.Now(),
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"order_id": saga.OrderID,
			"reason":   saga.FailureReason,
		},
	}

	return s.publisher.PublishSagaEvent(event)
}

func (s *SagaOrchestrator) sendCompensationEvent(saga *domain.SagaInstance, compensationStep domain.SagaStep) error {
	var event events.SagaEvent

	switch compensationStep {
	case domain.StepShippingCancelled:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "shipping.cancel",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"shipment_id": saga.Context["shipment_id"],
				"reason":      saga.FailureReason,
			},
		}

	case domain.StepInventoryReleased:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "inventory.release",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"reservation_ids": saga.Context["reservation_ids"],
				"reason":          saga.FailureReason,
			},
		}

	case domain.StepPaymentRefunded:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "payment.refund",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"payment_id":     saga.Context["payment_id"],
				"transaction_id": saga.Context["transaction_id"],
				"amount":         saga.Context["total_amount"],
				"reason":         saga.FailureReason,
			},
		}

	case domain.StepOrderCancelled:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "order.cancel",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"order_id": saga.OrderID,
				"reason":   saga.FailureReason,
			},
		}

	default:
		return fmt.Errorf("unknown compensation step: %s", compensationStep)
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("compensation event publish error: %v", err)
	}

	log.Printf("Compensation event is sent: %s -> %s", compensationStep, event.EventType)
	return nil
}

func (s *SagaOrchestrator) sendStepEvent(saga *domain.SagaInstance, step domain.SagaStep) error {
	var event events.SagaEvent

	switch step {
	case domain.StepPaymentProcessed:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "payment.process",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"order_id":       saga.OrderID,
				"customer_id":    saga.CustomerID,
				"amount":         saga.Context["total_amount"],
				"payment_method": "credit_card",
			},
		}

	case domain.StepInventoryReserved:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "inventory.reserve",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"order_id": saga.OrderID,
				"items":    saga.Context["items"],
			},
		}

	case domain.StepShippingCreated:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "shipping.create",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"order_id":    saga.OrderID,
				"customer_id": saga.CustomerID,
				"items":       saga.Context["items"],
			},
		}

	case domain.StepNotificationSent:
		event = events.SagaEvent{
			ID:            uuid.New(),
			SagaID:        saga.ID,
			OrderID:       saga.OrderID,
			EventType:     "notification.send",
			Service:       "saga-orchestrator",
			Timestamp:     time.Now(),
			CorrelationID: uuid.New(),
			Payload: map[string]interface{}{
				"order_id":    saga.OrderID,
				"customer_id": saga.CustomerID,
				"type":        "order_confirmation",
				"message":     "Siparişiniz başarıyla oluşturuldu!",
			},
		}

	default:
		return fmt.Errorf("unknown step: %s", step)
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("step event publish error: %v", err)
	}

	log.Printf("Step event sent: %s -> %s", step, event.EventType)
	return nil
}
