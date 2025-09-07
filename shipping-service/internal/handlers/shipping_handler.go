package handlers

import (
	"fmt"
	"log"

	"github.com/distributed-ecommerce-saga/shared-domain/events"
	sharedHTTP "github.com/distributed-ecommerce-saga/shared-domain/http"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shared-domain/types"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/domain"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShippingHandler struct {
	shippingService *service.ShippingService
}

func NewShippingHandler(shippingService *service.ShippingService) *ShippingHandler {
	return &ShippingHandler{
		shippingService: shippingService,
	}
}

func (h *ShippingHandler) HealthCheck(c *fiber.Ctx) error {
	return sharedHTTP.SuccessResponse(c, "Shipping service is healthy", map[string]interface{}{
		"service": "shipping-service",
		"status":  "healthy",
	})
}

func (h *ShippingHandler) GetShipmentByOrderID(c *fiber.Ctx) error {
	orderIDStr := c.Params("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return sharedHTTP.BadRequestResponse(c, "Invalid order ID", map[string]interface{}{
			"order_id": orderIDStr,
		})
	}

	shipment, err := h.shippingService.GetShipmentByOrderID(orderID)
	if err != nil {
		return sharedHTTP.NotFoundResponse(c, "Shipment not found")
	}

	response := ShipmentResponse{
		ID:            shipment.ID,
		OrderID:       shipment.OrderID,
		CustomerID:    shipment.CustomerID,
		SagaID:        shipment.SagaID,
		Status:        string(shipment.Status),
		TrackingID:    shipment.TrackingID,
		Address:       shipment.Address,
		FailureReason: shipment.FailureReason,
		CreatedAt:     shipment.CreatedAt,
		UpdatedAt:     shipment.UpdatedAt,
	}

	return sharedHTTP.SuccessResponse(c, "Shipment retrieved successfully", response)
}

func (h *ShippingHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Shipping service saga event received: %s from %s",
		event.EventType, event.Service)

	switch event.EventType {
	case "shipping.create":
		return h.handleShippingCreateCommand(event)

	case "shipping.cancel":
		return h.handleShippingCancelCommand(event)

	default:
		log.Printf("Unhandled event type: %s", event.EventType)
		return nil
	}
}

func (h *ShippingHandler) handleShippingCreateCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for shipping.create", event)
	}

	request, err := h.mapToShippingCreateRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Payload mapping error: %v", err), event)
	}

	if err := h.shippingService.CreateShipment(request); err != nil {
		log.Printf("Shipping create error: %v", err)
		return err
	}

	return nil
}

func (h *ShippingHandler) handleShippingCancelCommand(event events.SagaEvent) error {
	payloadMap, ok := event.Payload.(map[string]interface{})
	if !ok {
		return h.logAndReturnError("Invalid payload format for shipping.cancel", event)
	}

	request, err := h.mapToShippingCancelRequest(event.SagaID, payloadMap)
	if err != nil {
		return h.logAndReturnError(fmt.Sprintf("Cancel payload mapping error: %v", err), event)
	}

	if err := h.shippingService.CancelShipment(request); err != nil {
		log.Printf("Shipping cancel error: %v", err)
		return err
	}

	return nil
}

func (h *ShippingHandler) mapToShippingCreateRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.ShippingCreateRequest, error) {
	request := domain.ShippingCreateRequest{
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

	if itemsData, ok := payload["items"].([]interface{}); ok {
		for _, itemData := range itemsData {
			if itemMap, ok := itemData.(map[string]interface{}); ok {
				item := domain.ShippingItem{}

				if productIDStr, ok := itemMap["product_id"].(string); ok {
					if productID, err := uuid.Parse(productIDStr); err == nil {
						item.ProductID = productID
					}
				}

				if quantity, ok := itemMap["quantity"].(float64); ok {
					item.Quantity = int(quantity)
				}

				item.Weight = 1.0
				request.Items = append(request.Items, item)
			}
		}
	}

	if addressData, ok := payload["address"].(map[string]interface{}); ok {
		request.Address = types.ShippingAddress{
			Street:  getStringFromPayload(addressData, "street"),
			City:    getStringFromPayload(addressData, "city"),
			State:   getStringFromPayload(addressData, "state"),
			ZipCode: getStringFromPayload(addressData, "zip_code"),
			Country: getStringFromPayload(addressData, "country"),
		}
	}

	return request, nil
}

func (h *ShippingHandler) mapToShippingCancelRequest(sagaID uuid.UUID, payload map[string]interface{}) (domain.ShippingCancelRequest, error) {
	request := domain.ShippingCancelRequest{
		SagaID: sagaID,
	}

	if orderIDStr, ok := payload["order_id"].(string); ok {
		if orderID, err := uuid.Parse(orderIDStr); err == nil {
			request.OrderID = orderID
		}
	}

	if reason, ok := payload["reason"].(string); ok {
		request.Reason = reason
	}

	if shipmentIDStr, ok := payload["shipment_id"].(string); ok {
		if shipmentID, err := uuid.Parse(shipmentIDStr); err == nil {
			request.ShipmentID = shipmentID
		}
	}

	return request, nil
}

func getStringFromPayload(payload map[string]interface{}, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

func (h *ShippingHandler) logAndReturnError(message string, event events.SagaEvent) error {
	log.Printf("%s - Event: %+v", message, event)
	return fmt.Errorf(message)
}

func (h *ShippingHandler) StartConsuming(consumer *messaging.Consumer) error {
	routingKeys := []string{
		"saga.saga-orchestrator.shipping.create",
		"saga.saga-orchestrator.shipping.cancel",
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}
