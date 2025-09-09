package repository

import (
	"database/sql"
	"fmt"
	"github.com/distributed-ecommerce-saga/shared-domain/types"

	"github.com/distributed-ecommerce-saga/payment-service/internal/domain"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) CreatePayment(payment *domain.PaymentAggregate) error {
	query := `
		INSERT INTO payments (
			id, order_id, customer_id, saga_id, amount, payment_method, 
			status, transaction_id, external_ref, failure_reason, 
			refunded_amount, refund_reference, created_at, updated_at, 
			processed_at, refunded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.Exec(
		query,
		payment.ID,
		payment.OrderID,
		payment.CustomerID,
		payment.SagaID,
		payment.Amount,
		payment.PaymentMethod,
		payment.Status,
		payment.TransactionID,
		payment.ExternalRef,
		payment.FailureReason,
		payment.RefundedAmount,
		payment.RefundReference,
		payment.CreatedAt,
		payment.UpdatedAt,
		payment.ProcessedAt,
		payment.RefundedAt,
	)

	if err != nil {
		return fmt.Errorf("payment create error: %v", err)
	}

	return nil
}

// UpdatePayment mevcut payment'i günceller
func (r *PaymentRepository) UpdatePayment(payment *domain.PaymentAggregate) error {
	query := `
		UPDATE payments 
		SET status = $2, transaction_id = $3, external_ref = $4, 
			failure_reason = $5, refunded_amount = $6, refund_reference = $7,
			updated_at = $8, processed_at = $9, refunded_at = $10
		WHERE id = $1
	`

	result, err := r.db.Exec(
		query,
		payment.ID,
		payment.Status,
		payment.TransactionID,
		payment.ExternalRef,
		payment.FailureReason,
		payment.RefundedAmount,
		payment.RefundReference,
		payment.UpdatedAt,
		payment.ProcessedAt,
		payment.RefundedAt,
	)

	if err != nil {
		return fmt.Errorf("payment update hatası: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment not found: %s", payment.ID)
	}

	return nil
}

func (r *PaymentRepository) GetPaymentByID(paymentID uuid.UUID) (*domain.PaymentAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, amount, payment_method,
			   status, transaction_id, external_ref, failure_reason,
			   refunded_amount, refund_reference, created_at, updated_at,
			   processed_at, refunded_at
		FROM payments 
		WHERE id = $1
	`

	payment := &domain.PaymentAggregate{Payment: &types.Payment{}}
	var transactionID, externalRef, failureReason, refundRef sql.NullString
	var processedAt, refundedAt sql.NullTime

	err := r.db.QueryRow(query, paymentID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.CustomerID,
		&payment.SagaID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.Status,
		&transactionID,
		&externalRef,
		&failureReason,
		&payment.RefundedAmount,
		&refundRef,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&processedAt,
		&refundedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment not found: %s", paymentID)
		}
		return nil, fmt.Errorf("payment receive error: %v", err)
	}

	if transactionID.Valid {
		payment.TransactionID = transactionID.String
	}
	if externalRef.Valid {
		payment.ExternalRef = externalRef.String
	}
	if failureReason.Valid {
		payment.FailureReason = failureReason.String
	}
	if refundRef.Valid {
		payment.RefundReference = refundRef.String
	}
	if processedAt.Valid {
		payment.ProcessedAt = &processedAt.Time
	}
	if refundedAt.Valid {
		payment.RefundedAt = &refundedAt.Time
	}

	return payment, nil
}

func (r *PaymentRepository) GetPaymentByOrderID(orderID uuid.UUID) (*domain.PaymentAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, amount, payment_method,
			   status, transaction_id, external_ref, failure_reason,
			   refunded_amount, refund_reference, created_at, updated_at,
			   processed_at, refunded_at
		FROM payments 
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	payment := &domain.PaymentAggregate{Payment: &types.Payment{}}
	var transactionID, externalRef, failureReason, refundRef sql.NullString
	var processedAt, refundedAt sql.NullTime

	err := r.db.QueryRow(query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.CustomerID,
		&payment.SagaID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.Status,
		&transactionID,
		&externalRef,
		&failureReason,
		&payment.RefundedAmount,
		&refundRef,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&processedAt,
		&refundedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment not found for order: %s", orderID)
		}
		return nil, fmt.Errorf("payment receive hatası: %v", err)
	}

	// Nullable fields handling (aynı logic)
	if transactionID.Valid {
		payment.TransactionID = transactionID.String
	}
	if externalRef.Valid {
		payment.ExternalRef = externalRef.String
	}
	if failureReason.Valid {
		payment.FailureReason = failureReason.String
	}
	if refundRef.Valid {
		payment.RefundReference = refundRef.String
	}
	if processedAt.Valid {
		payment.ProcessedAt = &processedAt.Time
	}
	if refundedAt.Valid {
		payment.RefundedAt = &refundedAt.Time
	}

	return payment, nil
}

// GetPaymentsBySagaID Saga ID'ye göre tüm payment'ları getirir
func (r *PaymentRepository) GetPaymentsBySagaID(sagaID uuid.UUID) ([]*domain.PaymentAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, amount, payment_method,
			   status, transaction_id, external_ref, failure_reason,
			   refunded_amount, refund_reference, created_at, updated_at,
			   processed_at, refunded_at
		FROM payments 
		WHERE saga_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, sagaID)
	if err != nil {
		return nil, fmt.Errorf("payments receive hatası: %v", err)
	}
	defer rows.Close()

	var payments []*domain.PaymentAggregate

	for rows.Next() {
		payment := &domain.PaymentAggregate{Payment: &types.Payment{}}
		var transactionID, externalRef, failureReason, refundRef sql.NullString
		var processedAt, refundedAt sql.NullTime

		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.CustomerID,
			&payment.SagaID,
			&payment.Amount,
			&payment.PaymentMethod,
			&payment.Status,
			&transactionID,
			&externalRef,
			&failureReason,
			&payment.RefundedAmount,
			&refundRef,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&processedAt,
			&refundedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("payment scan error: %v", err)
		}

		// Nullable fields handling
		if transactionID.Valid {
			payment.TransactionID = transactionID.String
		}
		if externalRef.Valid {
			payment.ExternalRef = externalRef.String
		}
		if failureReason.Valid {
			payment.FailureReason = failureReason.String
		}
		if refundRef.Valid {
			payment.RefundReference = refundRef.String
		}
		if processedAt.Valid {
			payment.ProcessedAt = &processedAt.Time
		}
		if refundedAt.Valid {
			payment.RefundedAt = &refundedAt.Time
		}

		payments = append(payments, payment)
	}

	return payments, nil
}
