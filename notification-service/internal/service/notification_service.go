package service

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/distributed-ecommerce-saga/notification-service/internal/domain"
	"github.com/distributed-ecommerce-saga/notification-service/internal/repository"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/google/uuid"
)

type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	publisher        *messaging.Publisher
	failureRate      float64
}

func NewNotificationService(notificationRepo *repository.NotificationRepository, publisher *messaging.Publisher, failureRate float64) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		publisher:        publisher,
		failureRate:      failureRate,
	}
}

func (s *NotificationService) SendNotification(request domain.NotificationSendRequest) error {
	log.Printf("Notification send started: OrderID=%s, Type=%s", request.OrderID, request.Type)

	notificationType := types.NotificationTypeEmail
	switch request.Type {
	case "email":
		notificationType = types.NotificationTypeEmail
	case "sms":
		notificationType = types.NotificationTypeSMS
	case "push":
		notificationType = types.NotificationTypePush
	}

	notification := domain.NewNotificationAggregate(
		request.OrderID,
		request.CustomerID,
		request.SagaID,
		notificationType,
		request.Subject,
		request.Message,
		request.Recipient,
	)

	if err := s.notificationRepo.CreateNotification(notification); err != nil {
		return s.publishNotificationFailedEvent(request.SagaID, request.OrderID,
			fmt.Sprintf("Failed to create notification: %v", err))
	}

	time.Sleep(time.Millisecond * 200)

	if rand.Float64() < s.failureRate {
		notification.MarkAsFailed()
		s.notificationRepo.UpdateNotification(notification)

		return s.publishNotificationFailedEvent(request.SagaID, request.OrderID,
			"Notification provider unavailable")
	}

	notification.MarkAsSent()

	if err := s.notificationRepo.UpdateNotification(notification); err != nil {
		log.Printf("Notification status update error: %v", err)
	}

	log.Printf("Mock notification sent: Type=%s, Recipient=%s, Subject=%s",
		request.Type, request.Recipient, request.Subject)

	return s.publishNotificationSentEvent(notification)
}

func (s *NotificationService) GetNotificationsByOrderID(orderID uuid.UUID) ([]*domain.NotificationAggregate, error) {
	return s.notificationRepo.GetNotificationsByOrderID(orderID)
}

func (s *NotificationService) publishNotificationSentEvent(notification *domain.NotificationAggregate) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        notification.SagaID,
		OrderID:       notification.OrderID,
		EventType:     events.NotificationSentEvent,
		Service:       "notification-service",
		CorrelationID: uuid.New(),
		Payload: events.NotificationSentPayload{
			Notification: *notification.Notification,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("notification sent event publish error: %v", err)
	}

	log.Printf("Notification sent event published: OrderID=%s, Type=%s",
		notification.OrderID, notification.Type)
	return nil
}

func (s *NotificationService) publishNotificationFailedEvent(sagaID, orderID uuid.UUID, reason string) error {
	event := events.SagaEvent{
		ID:            uuid.New(),
		SagaID:        sagaID,
		OrderID:       orderID,
		EventType:     events.NotificationFailedEvent,
		Service:       "notification-service",
		CorrelationID: uuid.New(),
		Payload: events.NotificationFailedPayload{
			OrderID: orderID,
			Reason:  reason,
		},
	}

	if err := s.publisher.PublishSagaEvent(event); err != nil {
		return fmt.Errorf("notification failed event publish error: %v", err)
	}

	log.Printf("Notification failed event published: OrderID=%s, Reason=%s", orderID, reason)
	return nil
}
