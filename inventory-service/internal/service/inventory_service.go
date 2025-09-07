package service

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/inventory-service/internal/domain"
	"github.com/distributed-ecommerce-saga/inventory-service/internal/repository"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type InventoryService struct {
	inventoryRepo *repository.InventoryRepository
	publisher     *messaging.Publisher
}

func NewInventoryService(inventoryRepo *repository.InventoryRepository, publisher *messaging.Publisher) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		publisher:     publisher,
	}
}

func (s *InventoryService) ReserveInventory(request domain.InventoryReserveRequest) error {
	log.Printf("Inventory reserve started: OrderID=%s", request.OrderID)

	var reservations []*domain.ReservationAggregate

	for _, item := range request.Items {
		product, err := s.inventoryRepo.GetProductByID(item.ProductID)
		if err != nil {
			return s.publishInventoryFailedEvent(request.SagaID, request.OrderID,
				item.ProductID, fmt.Sprintf("Product not found: %v", err))
		}

		if !product.CanReserve(item.Quantity) {
			return s.publishInventoryFailedEvent(request.SagaID, request.OrderID,
				item.ProductID, "Insufficient stock")
		}

		if err := product.Reserve(item.Quantity); err != nil {
			return s.publishInventoryFailedEvent(request.SagaID, request.OrderID,
				item.ProductID, err.Error())
		}

		if err := s.inventoryRepo.UpdateProduct(product); err != nil {
			return s.publishInventoryFailedEvent(request.SagaID, request.OrderID,
				item.ProductID, fmt.Sprintf("Failed to update product: %v", err))
		}

		reservation := domain.NewReservationAggregate(request.OrderID, item.ProductID, request.SagaID, item.Quantity)
		if err := s.inventoryRepo.CreateReservation(reservation); err != nil {
			return s.publishInventoryFailedEvent(request.SagaID, request.OrderID,
				item.ProductID, fmt.Sprintf("Failed to create reservation: %v", err))
		}

		reservations = append(reservations, reservation)
	}

	return s.publishInventoryReservedEvent(request.SagaID, request.OrderID, reservations)
}

func (s *InventoryService) ReleaseInventory(request domain.InventoryReleaseRequest) error {
	log.Printf("Inventory release started: OrderID=%s", request.OrderID)

	reservations, err := s.inventoryRepo.GetReservationsBySagaID(request.SagaID)
	if err != nil {
		return s.publishInventoryReleaseFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Failed to get reservations: %v", err))
	}

	for _, reservation := range reservations {
		product, err := s.inventoryRepo.GetProductByID(reservation.ProductID)
		if err != nil {
			log.Printf("Failed to get product for release: %v", err)
			continue
		}

		product.Release(reservation.Quantity)
		s.inventoryRepo.UpdateProduct(product)

		reservation.Release()
		s.inventoryRepo.UpdateReservation(reservation)
	}

	return s.publishInventoryReleasedEvent(request.SagaID, request.OrderID, reservations)
}

func (s *InventoryService) publishInventoryReservedEvent(sagaID, orderID uuid.UUID, reservations []*domain.ReservationAggregate) error {
	var reservationData []types.InventoryReservation
	for _, r := range reservations {
		reservationData = append(reservationData, *r.InventoryReservation)
	}

	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.InventoryReservedEvent,
		Service:       "inventory-service",
		CorrelationID: uuid.New(),
		Payload: events.InventoryReservedPayload{
			Reservations: reservationData,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("inventory reserved event publish error: %v", err)
	}

	log.Printf("Inventory reserved event published: OrderID=%s", orderID)
	return nil
}

func (s *InventoryService) publishInventoryFailedEvent(sagaID, orderID, productID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.InventoryFailedEvent,
		Service:       "inventory-service",
		CorrelationID: uuid.New(),
		Payload: events.InventoryFailedPayload{
			OrderID:   orderID,
			ProductID: productID,
			Reason:    reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("inventory failed event publish error: %v", err)
	}

	log.Printf("Inventory failed event published: OrderID=%s, Reason=%s", orderID, reason)
	return nil
}

func (s *InventoryService) publishInventoryReleasedEvent(sagaID, orderID uuid.UUID, reservations []*domain.ReservationAggregate) error {
	var reservationIDs []uuid.UUID
	for _, r := range reservations {
		reservationIDs = append(reservationIDs, r.ID)
	}

	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.InventoryReleasedEvent,
		Service:       "inventory-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"order_id":        orderID,
			"reservation_ids": reservationIDs,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("inventory released event publish error: %v", err)
	}

	log.Printf("Inventory released event published: OrderID=%s", orderID)
	return nil
}

func (s *InventoryService) publishInventoryReleaseFailedEvent(sagaID, orderID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     "inventory.release.failed",
		Service:       "inventory-service",
		CorrelationID: uuid.New(),
		Payload: map[string]interface{}{
			"reason": reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("inventory release failed event publish error: %v", err)
	}

	log.Printf("Inventory release failed event published: OrderID=%s, Reason=%s", orderID, reason)
	return nil
}
