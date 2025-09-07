package domain

import (
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type NotificationAggregate struct {
	*types.Notification
	SagaID uuid.UUID `json:"saga_id" db:"saga_id"`
}

func NewNotificationAggregate(orderID, customerID, sagaID uuid.UUID, notificationType types.NotificationType, subject, message, recipient string) *NotificationAggregate {
	return &NotificationAggregate{
		Notification: &types.Notification{
			ID:         uuid.New(),
			OrderID:    orderID,
			CustomerID: customerID,
			Type:       notificationType,
			Status:     types.NotificationStatusPending,
			Subject:    subject,
			Message:    message,
			Recipient:  recipient,
			CreatedAt:  time.Now(),
		},
		SagaID: sagaID,
	}
}

func (n *NotificationAggregate) MarkAsSent() {
	n.Status = types.NotificationStatusSent
	now := time.Now()
	n.SentAt = &now
}

func (n *NotificationAggregate) MarkAsFailed() {
	n.Status = types.NotificationStatusFailed
}

type NotificationSendRequest struct {
	SagaID     uuid.UUID `json:"saga_id"`
	OrderID    uuid.UUID `json:"order_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Type       string    `json:"type"`
	Subject    string    `json:"subject"`
	Message    string    `json:"message"`
	Recipient  string    `json:"recipient"`
}
