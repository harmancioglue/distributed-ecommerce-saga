package domain

import (
	"fmt"
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type InventoryAggregate struct {
	*types.Product
	SagaID uuid.UUID `json:"saga_id" db:"saga_id"`
}

type ReservationAggregate struct {
	*types.InventoryReservation
	SagaID uuid.UUID `json:"saga_id" db:"saga_id"`
}

func NewReservationAggregate(orderID, productID, sagaID uuid.UUID, quantity int) *ReservationAggregate {
	return &ReservationAggregate{
		InventoryReservation: &types.InventoryReservation{
			ID:         uuid.New(),
			OrderID:    orderID,
			ProductID:  productID,
			Quantity:   quantity,
			Status:     types.InventoryStatusReserved,
			ReservedAt: time.Now(),
			ExpiresAt:  time.Now().Add(time.Hour * 24),
			UpdatedAt:  time.Now(),
		},
		SagaID: sagaID,
	}
}

func (r *ReservationAggregate) Release() {
	r.Status = types.InventoryStatusReleased
	r.UpdatedAt = time.Now()
}

func (r *ReservationAggregate) Complete() {
	r.Status = types.InventoryStatusSold
	r.UpdatedAt = time.Now()
}

func (i *InventoryAggregate) CanReserve(quantity int) bool {
	availableStock := i.Stock - i.ReservedStock
	return availableStock >= quantity
}

func (i *InventoryAggregate) Reserve(quantity int) error {
	if !i.CanReserve(quantity) {
		return fmt.Errorf("insufficient stock: available=%d, requested=%d",
			i.Stock-i.ReservedStock, quantity)
	}
	i.ReservedStock += quantity
	return nil
}

func (i *InventoryAggregate) Release(quantity int) {
	if i.ReservedStock >= quantity {
		i.ReservedStock -= quantity
	}
}

type InventoryReserveRequest struct {
	SagaID  uuid.UUID         `json:"saga_id"`
	OrderID uuid.UUID         `json:"order_id"`
	Items   []ReservationItem `json:"items"`
}

type ReservationItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type InventoryReleaseRequest struct {
	SagaID         uuid.UUID   `json:"saga_id"`
	OrderID        uuid.UUID   `json:"order_id"`
	ReservationIDs []uuid.UUID `json:"reservation_ids"`
}
