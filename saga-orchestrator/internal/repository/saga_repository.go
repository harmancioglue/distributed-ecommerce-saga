package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/domain"
	"github.com/google/uuid"
)

type SagaRepository struct {
	db *sql.DB
}

func NewSagaRepository(db *sql.DB) *SagaRepository {
	return &SagaRepository{
		db: db,
	}
}

func (r *SagaRepository) CreateSaga(saga *domain.SagaInstance) error {

	contextJson, err := json.Marshal(saga.Context)
	if err != nil {
		return fmt.Errorf("context serialization error: %v", err)
	}

	stepsJSON, err := json.Marshal(saga.CompletedSteps)
	if err != nil {
		return fmt.Errorf("steps serialization error: %v", err)
	}

	query := `
		INSERT INTO saga_instances (
			id, order_id, customer_id, status, current_step, 
			completed_steps, failure_reason, context, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Exec(
		query,
		saga.ID,
		saga.OrderID,
		saga.CustomerID,
		saga.Status,
		saga.CurrentStep,
		stepsJSON,
		saga.FailureReason,
		contextJson,
		saga.CreatedAt,
		saga.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("saga creation error: %v", err)
	}

	return nil
}

func (r *SagaRepository) GetSagaByID(sagaID uuid.UUID) (*domain.SagaInstance, error) {
	query := `
		SELECT id, order_id, customer_id, status, current_step, completed_steps,
			   failure_reason, context, created_at, updated_at, completed_at
		FROM saga_instances 
		WHERE id = $1
	`

	saga := &domain.SagaInstance{}
	var contextJSON, stepsJSON []byte
	var completedAt sql.NullTime

	err := r.db.QueryRow(query, sagaID).Scan(
		&saga.ID,
		&saga.OrderID,
		&saga.CustomerID,
		&saga.Status,
		&saga.CurrentStep,
		&stepsJSON,
		&saga.FailureReason,
		&contextJSON,
		&saga.CreatedAt,
		&saga.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("saga not found: %s", sagaID)
		}
		return nil, fmt.Errorf("saga receive error: %v", err)
	}

	// JSON'ları deserialize et
	if err := json.Unmarshal(contextJSON, &saga.Context); err != nil {
		return nil, fmt.Errorf("context deserialization error: %v", err)
	}

	if err := json.Unmarshal(stepsJSON, &saga.CompletedSteps); err != nil {
		return nil, fmt.Errorf("steps deserialization error: %v", err)
	}

	if completedAt.Valid {
		saga.CompletedAt = &completedAt.Time
	}

	return saga, nil
}

func (r *SagaRepository) UpdateSaga(saga *domain.SagaInstance) error {
	contextJSON, err := json.Marshal(saga.Context)
	if err != nil {
		return fmt.Errorf("context serialization error: %v", err)
	}

	stepsJSON, err := json.Marshal(saga.CompletedSteps)
	if err != nil {
		return fmt.Errorf("steps serialization error: %v", err)
	}

	query := `
		UPDATE saga_instances 
		SET status = $2, current_step = $3, completed_steps = $4, 
			failure_reason = $5, context = $6, updated_at = $7, completed_at = $8
		WHERE id = $1
	`

	result, err := r.db.Exec(
		query,
		saga.ID,
		saga.Status,
		saga.CurrentStep,
		stepsJSON,
		saga.FailureReason,
		contextJSON,
		saga.UpdatedAt,
		saga.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("saga update error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("saga not found: %s", saga.ID)
	}

	return nil
}

func (r *SagaRepository) GetSagaByOrderID(orderID uuid.UUID) (*domain.SagaInstance, error) {
	query := `
		SELECT id, order_id, customer_id, status, current_step, completed_steps,
			   failure_reason, context, created_at, updated_at, completed_at
		FROM saga_instances 
		WHERE order_id = $1
	`

	saga := &domain.SagaInstance{}
	var contextJSON, stepsJSON []byte
	var completedAt sql.NullTime

	err := r.db.QueryRow(query, orderID).Scan(
		&saga.ID,
		&saga.OrderID,
		&saga.CustomerID,
		&saga.Status,
		&saga.CurrentStep,
		&stepsJSON,
		&saga.FailureReason,
		&contextJSON,
		&saga.CreatedAt,
		&saga.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("saga not found order: %s", orderID)
		}
		return nil, fmt.Errorf("saga receive error: %v", err)
	}

	// Deserialization
	if err := json.Unmarshal(contextJSON, &saga.Context); err != nil {
		return nil, fmt.Errorf("context deserialization error: %v", err)
	}

	if err := json.Unmarshal(stepsJSON, &saga.CompletedSteps); err != nil {
		return nil, fmt.Errorf("steps deserialization error: %v", err)
	}

	if completedAt.Valid {
		saga.CompletedAt = &completedAt.Time
	}

	return saga, nil
}

// for recovery
func (r *SagaRepository) GetInProgressSagas() ([]*domain.SagaInstance, error) {
	query := `
		SELECT id, order_id, customer_id, status, current_step, completed_steps,
			   failure_reason, context, created_at, updated_at, completed_at
		FROM saga_instances 
		WHERE status IN ('started', 'in_progress', 'compensating')
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("sagas received error: %v", err)
	}
	defer rows.Close()

	var sagas []*domain.SagaInstance

	for rows.Next() {
		saga := &domain.SagaInstance{}
		var contextJSON, stepsJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&saga.ID,
			&saga.OrderID,
			&saga.CustomerID,
			&saga.Status,
			&saga.CurrentStep,
			&stepsJSON,
			&saga.FailureReason,
			&contextJSON,
			&saga.CreatedAt,
			&saga.UpdatedAt,
			&completedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("saga scan error: %v", err)
		}

		// Deserialization
		if err := json.Unmarshal(contextJSON, &saga.Context); err != nil {
			return nil, fmt.Errorf("context deserialization error: %v", err)
		}

		if err := json.Unmarshal(stepsJSON, &saga.CompletedSteps); err != nil {
			return nil, fmt.Errorf("steps deserialization error: %v", err)
		}

		if completedAt.Valid {
			saga.CompletedAt = &completedAt.Time
		}

		sagas = append(sagas, saga)
	}

	return sagas, nil
}
