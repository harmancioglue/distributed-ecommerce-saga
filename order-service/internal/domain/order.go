package domain

import (
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	"time"
)

type OrderAggregate struct {
	*types.Order
	SagaID        uuid.UUID `json:"saga_id,omitempty" db:"saga_id"`
	FailureReason string    `json:"failure_reason,omitempty" db:"failure_reason"`
}

func NewOrderAggregate(customerID uuid.UUID, items []types.OrderItem, shippingAddress *types.ShippingAddress) *OrderAggregate {
	orderID := uuid.New()

	var totalAmount float64
	for _, item := range items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	return &OrderAggregate{
		Order: &types.Order{
			ID:              orderID,
			CustomerID:      customerID,
			Items:           items,
			TotalAmount:     totalAmount,
			ShippingAddress: shippingAddress,
			Status:          types.OrderStatusPending,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}

func (o *OrderAggregate) UpdateStatus(status types.OrderStatus) {
	o.Status = status
	o.UpdatedAt = time.Now()
}

func (o *OrderAggregate) AttachSaga(sagaID uuid.UUID) {
	o.SagaID = sagaID
	o.UpdatedAt = time.Now()
}

func (o *OrderAggregate) SetFailureReason(reason string) {
	o.FailureReason = reason
	o.UpdatedAt = time.Now()
}

// CanProcessSaga checks that saga can be started
func (o *OrderAggregate) CanProcessSaga() bool {
	return o.Status == types.OrderStatusPending && o.TotalAmount > 0
}

type CreateOrderRequest struct {
	CustomerID      uuid.UUID              `json:"customer_id" validate:"required"`
	Items           []OrderItemRequest     `json:"items" validate:"required,min=1"`
	ShippingAddress ShippingAddressRequest `json:"shipping_address" validate:"required"`
}

type OrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required,min=1"`
	Price     float64   `json:"price" validate:"required,min=0"`
}

type ShippingAddressRequest struct {
	Street  string `json:"street" validate:"required"`
	City    string `json:"city" validate:"required"`
	State   string `json:"state" validate:"required"`
	ZipCode string `json:"zip_code" validate:"required"`
	Country string `json:"country" validate:"required"`
}

// converts to domain model
func (r CreateOrderRequest) ToOrderItems() []types.OrderItem {
	items := make([]types.OrderItem, len(r.Items))
	for i, item := range r.Items {
		items[i] = types.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return items
}

// converts to domain model
func (r CreateOrderRequest) ToShippingAddress() *types.ShippingAddress {
	return &types.ShippingAddress{
		Street:  r.ShippingAddress.Street,
		City:    r.ShippingAddress.City,
		State:   r.ShippingAddress.State,
		ZipCode: r.ShippingAddress.ZipCode,
		Country: r.ShippingAddress.Country,
	}
}
