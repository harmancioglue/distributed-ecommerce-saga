package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/distributed-ecommerce-saga/order-service/internal/domain"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(order *domain.OrderAggregate) error {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("items serialization error: %v", err)
	}

	addressJSON, err := json.Marshal(order.ShippingAddress)
	if err != nil {
		return fmt.Errorf("shipping address serialization error: %v", err)
	}

	query := `
		INSERT INTO orders (
			id, customer_id, items, total_amount, status, 
			shipping_address, saga_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(
		query,
		order.ID,
		order.CustomerID,
		itemsJSON,
		order.TotalAmount,
		order.Status,
		order.FailureReason,
		addressJSON,
		order.SagaID,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("order creation error: %v", err)
	}

	return nil
}

func (r *OrderRepository) UpdateOrder(order *domain.OrderAggregate) error {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("items serialization error: %v", err)
	}

	addressJSON, err := json.Marshal(order.ShippingAddress)
	if err != nil {
		return fmt.Errorf("shipping address serialization error: %v", err)
	}

	query := `
		UPDATE orders 
		SET status = $2, items = $3, total_amount = $4, 
			shipping_address = $5, saga_id = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.Exec(
		query,
		order.ID,
		order.Status,
		itemsJSON,
		order.FailureReason,
		order.TotalAmount,
		addressJSON,
		order.SagaID,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("order update error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found: %s", order.ID)
	}

	return nil
}

func (r *OrderRepository) GetOrderByID(orderID uuid.UUID) (*domain.OrderAggregate, error) {
	query := `
		SELECT id, customer_id, items, total_amount, status,
			   shipping_address, saga_id, created_at, updated_at
		FROM orders 
		WHERE id = $1
	`

	order := &domain.OrderAggregate{Order: &types.Order{}}
	var itemsJSON, addressJSON []byte
	var sagaID sql.NullString

	err := r.db.QueryRow(query, orderID).Scan(
		&order.ID,
		&order.CustomerID,
		&itemsJSON,
		&order.TotalAmount,
		&order.Status,
		&addressJSON,
		&sagaID,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("order receive error: %v", err)
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("items deserialization error: %v", err)
	}

	if err := json.Unmarshal(addressJSON, &order.ShippingAddress); err != nil {
		return nil, fmt.Errorf("shipping address deserialization error: %v", err)
	}

	// SagaID nullable
	if sagaID.Valid {
		if parsedUUID, err := uuid.Parse(sagaID.String); err == nil {
			order.SagaID = parsedUUID
		}
	}

	return order, nil
}

func (r *OrderRepository) GetOrdersByCustomerID(customerID uuid.UUID) ([]*domain.OrderAggregate, error) {
	query := `
		SELECT id, customer_id, items, total_amount, status,
			   shipping_address, saga_id, created_at, updated_at
		FROM orders 
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, customerID)
	if err != nil {
		return nil, fmt.Errorf("orders retrieval error: %v", err)
	}
	defer rows.Close()

	var orders []*domain.OrderAggregate

	for rows.Next() {
		order := &domain.OrderAggregate{Order: &types.Order{}}
		var itemsJSON, addressJSON []byte
		var sagaID sql.NullString

		err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&itemsJSON,
			&order.TotalAmount,
			&order.Status,
			&addressJSON,
			&sagaID,
			&order.CreatedAt,
			&order.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("order scan error: %v", err)
		}

		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("items deserialization error: %v", err)
		}

		if err := json.Unmarshal(addressJSON, &order.ShippingAddress); err != nil {
			return nil, fmt.Errorf("shipping address deserialization error: %v", err)
		}

		if sagaID.Valid {
			if parsedUUID, err := uuid.Parse(sagaID.String); err == nil {
				order.SagaID = parsedUUID
			}
		}

		orders = append(orders, order)
	}

	return orders, nil
}
