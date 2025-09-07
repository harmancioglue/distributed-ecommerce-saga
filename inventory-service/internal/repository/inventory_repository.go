package repository

import (
	"database/sql"
	"fmt"

	"github.com/distributed-ecommerce-saga/inventory-service/internal/domain"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) CreateReservation(reservation *domain.ReservationAggregate) error {
	query := `
		INSERT INTO inventory_reservations (
			id, order_id, product_id, saga_id, quantity, status, 
			reserved_at, expires_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(
		query,
		reservation.ID,
		reservation.OrderID,
		reservation.ProductID,
		reservation.SagaID,
		reservation.Quantity,
		reservation.Status,
		reservation.ReservedAt,
		reservation.ExpiresAt,
		reservation.UpdatedAt,
	)

	return err
}

func (r *InventoryRepository) UpdateReservation(reservation *domain.ReservationAggregate) error {
	query := `
		UPDATE inventory_reservations 
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(query, reservation.ID, reservation.Status, reservation.UpdatedAt)
	return err
}

func (r *InventoryRepository) GetProductByID(productID uuid.UUID) (*domain.InventoryAggregate, error) {
	query := `
		SELECT id, name, price, stock, reserved_stock
		FROM products 
		WHERE id = $1
	`

	product := &domain.InventoryAggregate{Product: &types.Product{}}
	err := r.db.QueryRow(query, productID).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
		&product.ReservedStock,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	return product, err
}

func (r *InventoryRepository) UpdateProduct(product *domain.InventoryAggregate) error {
	query := `
		UPDATE products 
		SET stock = $2, reserved_stock = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(query, product.ID, product.Stock, product.ReservedStock)
	return err
}

func (r *InventoryRepository) GetReservationsBySagaID(sagaID uuid.UUID) ([]*domain.ReservationAggregate, error) {
	query := `
		SELECT id, order_id, product_id, saga_id, quantity, status,
			   reserved_at, expires_at, updated_at
		FROM inventory_reservations
		WHERE saga_id = $1
	`

	rows, err := r.db.Query(query, sagaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []*domain.ReservationAggregate
	for rows.Next() {
		reservation := &domain.ReservationAggregate{
			InventoryReservation: &types.InventoryReservation{},
		}

		err := rows.Scan(
			&reservation.ID,
			&reservation.OrderID,
			&reservation.ProductID,
			&reservation.SagaID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.ReservedAt,
			&reservation.ExpiresAt,
			&reservation.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		reservations = append(reservations, reservation)
	}

	return reservations, nil
}
