package repository

import (
	"database/sql"

	"github.com/distributed-ecommerce-saga/notification-service/internal/domain"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) CreateNotification(notification *domain.NotificationAggregate) error {
	query := `
		INSERT INTO notifications (
			id, order_id, customer_id, saga_id, type, status, 
			subject, message, recipient, created_at, sent_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(
		query,
		notification.ID,
		notification.OrderID,
		notification.CustomerID,
		notification.SagaID,
		notification.Type,
		notification.Status,
		notification.Subject,
		notification.Message,
		notification.Recipient,
		notification.CreatedAt,
		notification.SentAt,
	)

	return err
}

func (r *NotificationRepository) UpdateNotification(notification *domain.NotificationAggregate) error {
	query := `
		UPDATE notifications 
		SET status = $2, sent_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(query, notification.ID, notification.Status, notification.SentAt)
	return err
}

func (r *NotificationRepository) GetNotificationsByOrderID(orderID uuid.UUID) ([]*domain.NotificationAggregate, error) {
	query := `
		SELECT id, order_id, customer_id, saga_id, type, status,
			   subject, message, recipient, created_at, sent_at
		FROM notifications 
		WHERE order_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.NotificationAggregate

	for rows.Next() {
		notification := &domain.NotificationAggregate{Notification: &types.Notification{}}
		var sentAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.OrderID,
			&notification.CustomerID,
			&notification.SagaID,
			&notification.Type,
			&notification.Status,
			&notification.Subject,
			&notification.Message,
			&notification.Recipient,
			&notification.CreatedAt,
			&sentAt,
		)

		if err != nil {
			return nil, err
		}

		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}
