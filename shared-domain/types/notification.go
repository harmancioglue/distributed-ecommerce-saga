package types

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

type Notification struct {
	ID         uuid.UUID          `json:"id"`
	OrderID    uuid.UUID          `json:"order_id"`
	CustomerID uuid.UUID          `json:"customer_id"`
	Type       NotificationType   `json:"type"`
	Status     NotificationStatus `json:"status"`
	Subject    string             `json:"subject"`
	Message    string             `json:"message"`
	Recipient  string             `json:"recipient"`
	CreatedAt  time.Time          `json:"created_at"`
	SentAt     *time.Time         `json:"sent_at,omitempty"`
}
