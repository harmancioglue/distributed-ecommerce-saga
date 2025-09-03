package types

import (
	"time"

	"github.com/google/uuid"
)

type InventoryStatus string

const (
	InventoryStatusAvailable InventoryStatus = "available"
	InventoryStatusReserved  InventoryStatus = "reserved"
	InventoryStatusReleased  InventoryStatus = "released"
	InventoryStatusSold      InventoryStatus = "sold"
)

type InventoryReservation struct {
	ID         uuid.UUID       `json:"id"`
	OrderID    uuid.UUID       `json:"order_id"`
	ProductID  uuid.UUID       `json:"product_id"`
	Quantity   int             `json:"quantity"`
	Status     InventoryStatus `json:"status"`
	ReservedAt time.Time       `json:"reserved_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type Product struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Stock         int       `json:"stock"`
	ReservedStock int       `json:"reserved_stock"`
}
