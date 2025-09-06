package handlers

import (
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type OrderResponse struct {
	ID              uuid.UUID               `json:"id"`
	CustomerID      uuid.UUID               `json:"customer_id"`
	Items           []OrderItemResponse     `json:"items"`
	TotalAmount     float64                 `json:"total_amount"`
	Status          string                  `json:"status"`
	ShippingAddress ShippingAddressResponse `json:"shipping_address"`
	SagaID          uuid.UUID               `json:"saga_id,omitempty"`
	FailureReason   string                  `json:"failure_reason,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

type OrderItemResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
}

type ShippingAddressResponse struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

func mapOrderItems(items []types.OrderItem) []OrderItemResponse {
	responses := make([]OrderItemResponse, len(items))
	for i, item := range items {
		responses[i] = OrderItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return responses
}

func mapShippingAddress(address *types.ShippingAddress) ShippingAddressResponse {
	if address == nil {
		return ShippingAddressResponse{}
	}
	return ShippingAddressResponse{
		Street:  address.Street,
		City:    address.City,
		State:   address.State,
		ZipCode: address.ZipCode,
		Country: address.Country,
	}
}
