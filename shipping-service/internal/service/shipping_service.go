package service

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/domain"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/repository"
	"github.com/google/uuid"
)

type ShippingService struct {
	shippingRepo *repository.ShippingRepository
	publisher    *messaging.Publisher
	failureRate  float64
}

func NewShippingService(shippingRepo *repository.ShippingRepository, publisher *messaging.Publisher, failureRate float64) *ShippingService {
	return &ShippingService{
		shippingRepo: shippingRepo,
		publisher:    publisher,
		failureRate:  failureRate,
	}
}

func (s *ShippingService) CreateShipment(request domain.ShippingCreateRequest) error {
	log.Printf("Shipping create started: OrderID=%s", request.OrderID)

	time.Sleep(time.Millisecond * 300)

	if rand.Float64() < s.failureRate {
		return s.publishShippingFailedEvent(request.SagaID, request.OrderID,
			"Shipping provider unavailable")
	}

	shipment := domain.NewShippingAggregate(request.OrderID, request.CustomerID, request.SagaID, request.Address)
	shipment.CreateShipment()

	if err := s.shippingRepo.CreateShipment(shipment); err != nil {
		return s.publishShippingFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Failed to create shipment: %v", err))
	}

	return s.publishShippingCreatedEvent(shipment)
}

func (s *ShippingService) CancelShipment(request domain.ShippingCancelRequest) error {
	log.Printf("Shipping cancel started: OrderID=%s", request.OrderID)

	var shipment *domain.ShippingAggregate
	var err error

	if request.ShipmentID != uuid.Nil {
		shipments, err := s.shippingRepo.GetShipmentsBySagaID(request.SagaID)
		if err == nil && len(shipments) > 0 {
			shipment = shipments[0]
		}
	} else {
		shipment, err = s.shippingRepo.GetShipmentByOrderID(request.OrderID)
	}

	if err != nil {
		return s.publishShippingCancelFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Shipment not found: %v", err))
	}

	if !shipment.CanCancel() {
		return s.publishShippingCancelFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Cannot cancel shipment in status: %s", shipment.Status))
	}

	shipment.CancelShipment(request.Reason)

	if err := s.shippingRepo.UpdateShipment(shipment); err != nil {
		return s.publishShippingCancelFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Failed to cancel shipment: %v", err))
	}

	return s.publishShippingCancelledEvent(shipment)
}

func (s *ShippingService) GetShipmentByOrderID(orderID uuid.UUID) (*domain.ShippingAggregate, error) {
	return s.shippingRepo.GetShipmentByOrderID(orderID)
}

func (s *ShippingService) publishShippingCreatedEvent(shipment *domain.ShippingAggregate) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        shipment.SagaID,
		OrderID:       shipment.OrderID,
		EventType:     events.ShippingCreatedEvent,
		Service:       "shipping-service",
		CorrelationID: uuid.New(),
		Payload: events.ShippingCreatedPayload{
			Shipment: *shipment.Shipment,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("shipping created event publish error: %v", err)
	}

	log.Printf("Shipping created event published: OrderID=%s, TrackingID=%s",
		shipment.OrderID, shipment.TrackingID)
	return nil
}

func (s *ShippingService) publishShippingFailedEvent(sagaID, orderID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.ShippingFailedEvent,
		Service:       "shipping-service",
		CorrelationID: uuid.New(),
		Payload: events.ShippingFailedPayload{
			OrderID: orderID,
			Reason:  reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("shipping failed event publish error: %v", err)
	}

	log.Printf("Shipping failed event published: OrderID=%s, Reason=%s", orderID, reason)
	return nil
}

func (s *ShippingService) publishShippingCancelledEvent(shipment *domain.ShippingAggregate) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        shipment.SagaID,
		OrderID:       shipment.OrderID,
		EventType:     events.ShippingCancelledEvent,
		Service:       "shipping-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"shipment_id":  shipment.ID,
			"tracking_id":  shipment.TrackingID,
			"cancelled_at": shipment.UpdatedAt,
			"reason":       shipment.FailureReason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("shipping cancelled event publish error: %v", err)
	}

	log.Printf("Shipping cancelled event published: OrderID=%s, TrackingID=%s",
		shipment.OrderID, shipment.TrackingID)
	return nil
}

func (s *ShippingService) publishShippingCancelFailedEvent(sagaID, orderID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     "shipping.cancel.failed",
		Service:       "shipping-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"reason": reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("shipping cancel failed event publish error: %v", err)
	}

	log.Printf("Shipping cancel failed event published: OrderID=%s, Reason=%s", orderID, reason)
	return nil
}
