package handlers

import (
	"log"

	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
)

type EventHandler struct {
	orchestrator *service.SagaOrchestrator
}

func NewEventHandler(orchestrator *service.SagaOrchestrator) *EventHandler {
	return &EventHandler{orchestrator: orchestrator}
}

func (h *EventHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Saga orchestrator event handler: %s from %s", event.EventType, event.Service)
	return h.orchestrator.ProcessIncomingEvent(event)
}

func (h *EventHandler) StartConsuming(consumer *messaging.Consumer) error {
	routingKeys := []string{
		"saga.order-service.*",        // Order service events
		"saga.payment-service.*",      // Payment service events
		"saga.inventory-service.*",    // Inventory service events
		"saga.shipping-service.*",     // Shipping service events
		"saga.notification-service.*", // Notification service events
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}
