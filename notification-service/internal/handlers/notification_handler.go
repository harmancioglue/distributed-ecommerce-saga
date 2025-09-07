package handlers

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/notification-service/internal/domain"
	"github.com/distributed-ecommerce-saga/notification-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	sharedHTTP "github.com/distributed-ecommerce-saga/shared-domain/http"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) HealthCheck(c *fiber.Ctx) error {
	return sharedHTTP.SuccessResponse(c, "Notification service is healthy", map[string]interface{}{
		"service": "notification-service",
		"status":  "healthy",
	})
}

func (h *NotificationHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Notification service saga event received: %s from %s",
		event.EventType, event.Service)

	switch event.EventType {
	case "notification.send":
		return h.handleNotificationSendCommand(event)

	default:
		log.Printf("Unhandled event type: %s", event.EventType)
		return nil
	}
}

func (h *NotificationHandler) handleNotificationSendCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for notification.send", event)
	}

	request, err := h.mapToNotificationSendRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Payload mapping error: %v", err), event)
	}

	if err := h.notificationService.SendNotification(request); err != nil {
		log.Printf("Notification send error: %v", err)
		return err
	}

	return nil
}

func (h *NotificationHandler) mapToNotificationSendRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.NotificationSendRequest, error) {
	request := domain.NotificationSendRequest{
		SagaID: sagaID,
	}

	if orderIDStr, ok := payload["order_id"].(string); ok {
		if orderID, err := uuid.Parse(orderIDStr); err == nil {
			request.OrderID = orderID
		}
	}

	if customerIDStr, ok := payload["customer_id"].(string); ok {
		if customerID, err := uuid.Parse(customerIDStr); err == nil {
			request.CustomerID = customerID
		}
	}

	if notType, ok := payload["type"].(string); ok {
		request.Type = notType
	} else {
		request.Type = "email"
	}

	if subject, ok := payload["subject"].(string); ok {
		request.Subject = subject
	}

	if message, ok := payload["message"].(string); ok {
		request.Message = message
	}

	if recipient, ok := payload["recipient"].(string); ok {
		request.Recipient = recipient
	} else {
		request.Recipient = "customer@example.com"
	}

	return request, nil
}

func (h *NotificationHandler) logAndReturnError(message string, event events.SagaEvent) error {
	log.Printf("%s - Event: %+v", message, event)
	return fmt.Errorf(message)
}

func (h *NotificationHandler) StartConsuming(consumer *messaging.Consumer) error {
	routingKeys := []string{
		"saga.saga-orchestrator.notification.send",
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}
