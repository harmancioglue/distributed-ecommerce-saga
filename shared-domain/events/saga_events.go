package events

import (
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type SagaEventType string

const (
	// Order Events
	OrderCreatedEvent   SagaEventType = "order.created"
	OrderCompletedEvent SagaEventType = "order.completed"
	OrderCancelledEvent SagaEventType = "order.cancelled"

	// Payment Events
	PaymentProcessedEvent SagaEventType = "payment.processed"
	PaymentFailedEvent    SagaEventType = "payment.failed"
	PaymentRefundedEvent  SagaEventType = "payment.refunded"

	// Inventory Events
	InventoryReservedEvent SagaEventType = "inventory.reserved"
	InventoryFailedEvent   SagaEventType = "inventory.failed"
	InventoryReleasedEvent SagaEventType = "inventory.released"

	// Shipping Events
	ShippingCreatedEvent   SagaEventType = "shipping.created"
	ShippingFailedEvent    SagaEventType = "shipping.failed"
	ShippingCancelledEvent SagaEventType = "shipping.cancelled"

	// Notification Events
	NotificationSentEvent   SagaEventType = "notification.sent"
	NotificationFailedEvent SagaEventType = "notification.failed"
)

type SagaEvent struct {
	ID            uuid.UUID     `json:"id"`
	SagaID        uuid.UUID     `json:"saga_id"`  // Saga instance ID
	OrderID       uuid.UUID     `json:"order_id"` // Business ID
	EventType     SagaEventType `json:"event_type"`
	Payload       interface{}   `json:"payload"`
	Timestamp     time.Time     `json:"timestamp"`
	Service       string        `json:"service"`        // Hangi servisten geldi
	CorrelationID uuid.UUID     `json:"correlation_id"` // Event tracking
}

type OrderCreatedPayload struct {
	Order types.Order `json:"order"`
}

type OrderCompletedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
	Status  string    `json:"status"`
}

type PaymentProcessedPayload struct {
	Payment types.Payment `json:"payment"`
}

type PaymentFailedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
	Reason  string    `json:"reason"`
	Amount  float64   `json:"amount"`
}

type InventoryReservedPayload struct {
	Reservations []types.InventoryReservation `json:"reservations"`
}

type InventoryFailedPayload struct {
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id"`
	Reason    string    `json:"reason"`
}

type ShippingCreatedPayload struct {
	Shipment types.Shipment `json:"shipment"`
}

type ShippingFailedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
	Reason  string    `json:"reason"`
}

type NotificationSentPayload struct {
	Notification types.Notification `json:"notification"`
}

type NotificationFailedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
	Reason  string    `json:"reason"`
}
