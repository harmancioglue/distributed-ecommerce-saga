package handlers

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/inventory-service/internal/domain"
	"github.com/distributed-ecommerce-saga/inventory-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	sharedHTTP "github.com/distributed-ecommerce-saga/shared-domain/http"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type InventoryHandler struct {
	inventoryService *service.InventoryService
}

func NewInventoryHandler(inventoryService *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

func (h *InventoryHandler) HealthCheck(c *fiber.Ctx) error {
	return sharedHTTP.SuccessResponse(c, "Inventory service is healthy", map[string]interface{}{
		"service": "inventory-service",
		"status":  "healthy",
	})
}

func (h *InventoryHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Inventory service saga event received: %s from %s",
		event.EventType, event.Service)

	switch event.EventType {
	case "inventory.reserve":
		return h.handleInventoryReserveCommand(event)

	case "inventory.release":
		return h.handleInventoryReleaseCommand(event)

	default:
		log.Printf("Unhandled event type: %s", event.EventType)
		return nil
	}
}

func (h *InventoryHandler) handleInventoryReserveCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for inventory.reserve", event)
	}

	request, err := h.mapToInventoryReserveRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Payload mapping error: %v", err), event)
	}

	if err := h.inventoryService.ReserveInventory(request); err != nil {
		log.Printf("Inventory reserve error: %v", err)
		return err
	}

	return nil
}

func (h *InventoryHandler) handleInventoryReleaseCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for inventory.release", event)
	}

	request, err := h.mapToInventoryReleaseRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Release payload mapping error: %v", err), event)
	}

	if err := h.inventoryService.ReleaseInventory(request); err != nil {
		log.Printf("Inventory release error: %v", err)
		return err
	}

	return nil
}

func (h *InventoryHandler) mapToInventoryReserveRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.InventoryReserveRequest, error) {
	request := domain.InventoryReserveRequest{
		SagaID: sagaID,
	}

	if orderIDStr, ok := payload["order_id"].(string); ok {
		if orderID, err := uuid.Parse(orderIDStr); err == nil {
			request.OrderID = orderID
		} else {
			return request, fmt.Errorf("invalid order_id format: %s", orderIDStr)
		}
	} else {
		return request, fmt.Errorf("missing or invalid order_id")
	}

	if itemsData, ok := payload["items"].([]interface{}); ok {
		for _, itemData := range itemsData {
			if itemMap, ok := itemData.(map[string]interface{}); ok {
				item := domain.ReservationItem{}

				if productIDStr, ok := itemMap["product_id"].(string); ok {
					if productID, err := uuid.Parse(productIDStr); err == nil {
						item.ProductID = productID
					}
				}

				if quantity, ok := itemMap["quantity"].(float64); ok {
					item.Quantity = int(quantity)
				}

				request.Items = append(request.Items, item)
			}
		}
	}

	return request, nil
}

func (h *InventoryHandler) mapToInventoryReleaseRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.InventoryReleaseRequest, error) {
	request := domain.InventoryReleaseRequest{
		SagaID: sagaID,
	}

	if orderIDStr, ok := payload["order_id"].(string); ok {
		if orderID, err := uuid.Parse(orderIDStr); err == nil {
			request.OrderID = orderID
		}
	}

	if reservationIDs, ok := payload["reservation_ids"].([]interface{}); ok {
		for _, idData := range reservationIDs {
			if idStr, ok := idData.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					request.ReservationIDs = append(request.ReservationIDs, id)
				}
			}
		}
	}

	return request, nil
}

func (h *InventoryHandler) logAndReturnError(message string, event events.SagaEvent) error {
	log.Printf("%s - Event: %+v", message, event)
	return fmt.Errorf(message)
}

func (h *InventoryHandler) StartConsuming(consumer *messaging.Consumer) error {
	routingKeys := []string{
		"saga.saga-orchestrator.inventory.reserve",
		"saga.saga-orchestrator.inventory.release",
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}
