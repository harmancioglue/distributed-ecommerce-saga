package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/domain"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type ShippingRepository struct {
	db *sql.DB
}

func NewShippingRepository(db *sql.DB) *ShippingRepository {
	return &ShippingRepository{db: db}
}

func (r *ShippingRepository) CreateShipment(shipment *domain.ShippingAggregate) error {
	addressJSON, err := json.Marshal(shipment.Address)
	if err != nil {
		return fmt.Errorf("address serialization error: %v", err)
	}

	query := `
		INSERT INTO shipments (
			id, order_id, customer_id, saga_id, status, tracking_id,
			address, failure_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Exec(
		query,
		shipment.ID,
		shipment.OrderID,
		shipment.CustomerID,
		shipment.SagaID,
		shipment.Status,
		shipment.TrackingID,
		addressJSON,
		shipment.FailureReason,
		shipment.CreatedAt,
		shipment.UpdatedAt,
	)

	return err
}

func (r *ShippingRepository) UpdateShipment(shipment *domain.ShippingAggregate) error {
	addressJSON, err := json.Marshal(shipment.Address)
	if err != nil {
		return fmt.Errorf("address serialization error: %v", err)
	}

	query := `
		UPDATE shipments 
		SET status = $2, tracking_id = $3, address = $4, 
			failure_reason = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.Exec(
		query,
		shipment.ID,
		shipment.Status,
		shipment.TrackingID,
		addressJSON,
		shipment.FailureReason,
		shipment.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("shipment not found: %s", shipment.ID)
	}

	return nil
}

func (r *ShippingRepository) GetShipmentByOrderID(orderID uuid.UUID) (*domain.ShippingAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, status, tracking_id,
			   address, failure_reason, created_at, updated_at
		FROM shipments 
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	shipment := &domain.ShippingAggregate{Shipment: &types.Shipment{}}
	var addressJSON []byte
	var failureReason sql.NullString

	err := r.db.QueryRow(query, orderID).Scan(
		&shipment.ID,
		&shipment.OrderID,
		&shipment.CustomerID,
		&shipment.SagaID,
		&shipment.Status,
		&shipment.TrackingID,
		&addressJSON,
		&failureReason,
		&shipment.CreatedAt,
		&shipment.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("shipment not found for order: %s", orderID)
		}
		return nil, fmt.Errorf("shipment retrieval error: %v", err)
	}

	if err := json.Unmarshal(addressJSON, &shipment.Address); err != nil {
		return nil, fmt.Errorf("address deserialization error: %v", err)
	}

	if failureReason.Valid {
		shipment.FailureReason = failureReason.String
	}

	return shipment, nil
}

func (r *ShippingRepository) GetShipmentsBySagaID(sagaID uuid.UUID) ([]*domain.ShippingAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, status, tracking_id,
			   address, failure_reason, created_at, updated_at
		FROM shipments 
		WHERE saga_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, sagaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []*domain.ShippingAggregate

	for rows.Next() {
		shipment := &domain.ShippingAggregate{Shipment: &types.Shipment{}}
		var addressJSON []byte
		var failureReason sql.NullString

		err := rows.Scan(
			&shipment.ID,
			&shipment.OrderID,
			&shipment.CustomerID,
			&shipment.SagaID,
			&shipment.Status,
			&shipment.TrackingID,
			&addressJSON,
			&failureReason,
			&shipment.CreatedAt,
			&shipment.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(addressJSON, &shipment.Address); err != nil {
			return nil, fmt.Errorf("address deserialization error: %v", err)
		}

		if failureReason.Valid {
			shipment.FailureReason = failureReason.String
		}

		shipments = append(shipments, shipment)
	}

	return shipments, nil
}
