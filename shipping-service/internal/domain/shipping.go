package domain

import (
	"fmt"
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type ShippingAggregate struct {
	*types.Shipment
	SagaID        uuid.UUID `json:"saga_id" db:"saga_id"`
	FailureReason string    `json:"failure_reason,omitempty" db:"failure_reason"`
}

func NewShippingAggregate(orderID, customerID, sagaID uuid.UUID, address types.ShippingAddress) *ShippingAggregate {
	return &ShippingAggregate{
		Shipment: &types.Shipment{
			ID:         uuid.New(),
			OrderID:    orderID,
			CustomerID: customerID,
			Address:    address,
			Status:     types.ShippingStatusPending,
			TrackingID: generateTrackingID(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		SagaID: sagaID,
	}
}

func (s *ShippingAggregate) CreateShipment() {
	s.Status = types.ShippingStatusPreparing
	s.UpdatedAt = time.Now()
}

func (s *ShippingAggregate) StartShipping() {
	s.Status = types.ShippingStatusShipped
	s.UpdatedAt = time.Now()
}

func (s *ShippingAggregate) CancelShipment(reason string) {
	s.Status = types.ShippingStatusCancelled
	s.FailureReason = reason
	s.UpdatedAt = time.Now()
}

func (s *ShippingAggregate) CompleteDelivery() {
	s.Status = types.ShippingStatusDelivered
	s.UpdatedAt = time.Now()
}

func (s *ShippingAggregate) CanCancel() bool {
	return s.Status == types.ShippingStatusPending || s.Status == types.ShippingStatusPreparing
}

func generateTrackingID() string {
	return fmt.Sprintf("TRK_%d", time.Now().Unix())
}

type ShippingCreateRequest struct {
	SagaID     uuid.UUID             `json:"saga_id"`
	OrderID    uuid.UUID             `json:"order_id"`
	CustomerID uuid.UUID             `json:"customer_id"`
	Items      []ShippingItem        `json:"items"`
	Address    types.ShippingAddress `json:"address"`
}

type ShippingItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Weight    float64   `json:"weight"`
}

type ShippingCancelRequest struct {
	SagaID     uuid.UUID `json:"saga_id"`
	OrderID    uuid.UUID `json:"order_id"`
	ShipmentID uuid.UUID `json:"shipment_id,omitempty"`
	Reason     string    `json:"reason"`
}
