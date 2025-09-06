package service

import (
	"fmt"
	"github.com/distributed-ecommerce-saga/order-service/internal/domain"
	"github.com/distributed-ecommerce-saga/order-service/internal/repository"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	"log"
)

type OrderService struct {
	orderRepo *repository.OrderRepository
	publisher *messaging.Publisher
}

func NewOrderService(orderRepo *repository.OrderRepository, publisher *messaging.Publisher) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

func (s *OrderService) CreateOrder(request domain.CreateOrderRequest) (*domain.OrderAggregate, error) {
	order := domain.NewOrderAggregate(
		request.CustomerID,
		request.ToOrderItems(),
		request.ToShippingAddress(),
	)

	if !order.CanProcessSaga() {
		return nil, fmt.Errorf("order is invalid for saga")
	}

	if err := s.orderRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("order creation error: %v", err)
	}

	log.Printf("Order created: OrderID=%s, CustomerID=%s, Amount=%.2f",
		order.ID, order.CustomerID, order.TotalAmount)

	// publish event for saga
	if err := s.publishOrderCreatedEvent(order); err != nil {
		// Order created but saga not started
		log.Printf("Saga creation error: %v", err)

		order.UpdateStatus(types.OrderStatusFailed)
		order.SetFailureReason(fmt.Sprintf("Saga creation error: %v", err))

		if updateErr := s.orderRepo.UpdateOrder(order); updateErr != nil {
			log.Printf("Order failed status update error: %v", updateErr)
		}

		return order, fmt.Errorf("saga creation error: %v", err)
	}

	return order, nil
}

func (s *OrderService) publishOrderCreatedEvent(order *domain.OrderAggregate) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        uuid.New(),
		OrderID:       order.ID,
		EventType:     events.OrderCreatedEvent,
		Service:       "order-service",
		CorrelationID: uuid.New(),
		Payload: events.OrderCreatedPayload{
			Order: *order.Order,
		},
	}

	order.AttachSaga(event.SagaID)
	order.UpdateStatus(types.OrderStatusProcessing)

	if err := s.orderRepo.UpdateOrder(order); err != nil {
		return fmt.Errorf("order saga ID update error: %v", err)
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("order created event publish error: %v", err)
	}

	log.Printf("Order created event published: SagaID=%s, OrderID=%s", event.SagaID, order.ID)
	return nil
}

func (s *OrderService) GetOrderByID(orderID uuid.UUID) (*domain.OrderAggregate, error) {
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}
	return order, nil
}

func (s *OrderService) GetOrdersByCustomerID(customerID uuid.UUID) ([]*domain.OrderAggregate, error) {
	orders, err := s.orderRepo.GetOrdersByCustomerID(customerID)
	if err != nil {
		return nil, fmt.Errorf("orders receive error: %v", err)
	}
	return orders, nil
}

func (s *OrderService) ProcessSagaCompletionEvent(event events.SagaEvent) error {
	order, err := s.orderRepo.GetOrderByID(event.OrderID)
	if err != nil {
		return fmt.Errorf("order not found: %v", err)
	}

	switch event.EventType {
	case events.OrderCompletedEvent:
		order.UpdateStatus(types.OrderStatusCompleted)
		log.Printf("Order completed successfully: OrderID=%s", order.ID)

	case events.OrderCancelledEvent:
		order.UpdateStatus(types.OrderStatusCancelled)
		if payload, ok := event.Payload.(map[string]interface{}); ok {
			if reason, exists := payload["reason"]; exists {
				if reasonStr, ok := reason.(string); ok {
					order.SetFailureReason(reasonStr)
				}
			}
		}
		log.Printf("Order is cancelled: OrderID=%s, Reason=%s", order.ID, order.FailureReason)

	default:
		return fmt.Errorf("unknown saga completion event: %s", event.EventType)
	}

	if err := s.orderRepo.UpdateOrder(order); err != nil {
		return fmt.Errorf("order status update error: %v", err)
	}

	return nil
}
