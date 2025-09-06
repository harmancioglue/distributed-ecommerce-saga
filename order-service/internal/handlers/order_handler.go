package handlers

import (
	"github.com/distributed-ecommerce-saga/order-service/internal/domain"
	"github.com/distributed-ecommerce-saga/order-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	sharedHTTP "github.com/distributed-ecommerce-saga/shared-domain/http"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log"
	"strconv"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) GetOrderByID(c *fiber.Ctx) error {
	orderIDStr := c.Params("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return sharedHTTP.BadRequestResponse(c, "Invalid order ID", map[string]interface{}{
			"order_id": orderIDStr,
		})
	}

	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		return sharedHTTP.NotFoundResponse(c, "Order not found")
	}

	response := OrderResponse{
		ID:              order.ID,
		CustomerID:      order.CustomerID,
		Items:           mapOrderItems(order.Items),
		TotalAmount:     order.TotalAmount,
		Status:          string(order.Status),
		ShippingAddress: mapShippingAddress(order.ShippingAddress),
		SagaID:          order.SagaID,
		FailureReason:   order.FailureReason,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}

	return sharedHTTP.SuccessResponse(c, "Order retrieved successfully", response)
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var request domain.CreateOrderRequest

	if err := c.BodyParser(&request); err != nil {
		return sharedHTTP.BadRequestResponse(c, "Invalid request body", map[string]interface{}{
			"parse_error": err.Error(),
		})
	}

	// Basic validation
	if request.CustomerID == uuid.Nil {
		return sharedHTTP.BadRequestResponse(c, "Customer ID is required", nil)
	}

	if len(request.Items) == 0 {
		return sharedHTTP.BadRequestResponse(c, "At least one item is required", nil)
	}

	for i, item := range request.Items {
		if item.ProductID == uuid.Nil {
			return sharedHTTP.BadRequestResponse(c, "Invalid product ID", map[string]interface{}{
				"item_index": i,
			})
		}
		if item.Quantity <= 0 {
			return sharedHTTP.BadRequestResponse(c, "Invalid quantity", map[string]interface{}{
				"item_index": i,
				"quantity":   item.Quantity,
			})
		}
		if item.Price < 0 {
			return sharedHTTP.BadRequestResponse(c, "Invalid price", map[string]interface{}{
				"item_index": i,
				"price":      item.Price,
			})
		}
	}

	order, err := h.orderService.CreateOrder(request)
	if err != nil {
		log.Printf("Order creation error: %v", err)
		return sharedHTTP.InternalServerErrorResponse(c, "Order creation failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Response DTO oluştur
	response := OrderResponse{
		ID:              order.ID,
		CustomerID:      order.CustomerID,
		Items:           mapOrderItems(order.Items),
		TotalAmount:     order.TotalAmount,
		Status:          string(order.Status),
		ShippingAddress: mapShippingAddress(order.ShippingAddress),
		SagaID:          order.SagaID,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}

	return sharedHTTP.CreatedResponse(c, "Order created successfully", response)
}

func (h *OrderHandler) GetOrdersByCustomerID(c *fiber.Ctx) error {
	customerIDStr := c.Params("customer_id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return sharedHTTP.BadRequestResponse(c, "Invalid customer ID", map[string]interface{}{
			"customer_id": customerIDStr,
		})
	}

	page := 1
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	orders, err := h.orderService.GetOrdersByCustomerID(customerID)
	if err != nil {
		return sharedHTTP.InternalServerErrorResponse(c, "Orders retrieval failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Pagination logic (basic version)
	start := (page - 1) * limit
	end := start + limit
	if start > len(orders) {
		start = len(orders)
	}
	if end > len(orders) {
		end = len(orders)
	}

	paginatedOrders := orders[start:end]
	responses := make([]OrderResponse, len(paginatedOrders))

	for i, order := range paginatedOrders {
		responses[i] = OrderResponse{
			ID:              order.ID,
			CustomerID:      order.CustomerID,
			Items:           mapOrderItems(order.Items),
			TotalAmount:     order.TotalAmount,
			Status:          string(order.Status),
			ShippingAddress: mapShippingAddress(order.ShippingAddress),
			SagaID:          order.SagaID,
			FailureReason:   order.FailureReason,
			CreatedAt:       order.CreatedAt,
			UpdatedAt:       order.UpdatedAt,
		}
	}

	return sharedHTTP.SuccessResponse(c, "Orders retrieved successfully", map[string]interface{}{
		"orders": responses,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    len(orders),
			"has_more": end < len(orders),
		},
	})
}

func (h *OrderHandler) HealthCheck(c *fiber.Ctx) error {
	return sharedHTTP.SuccessResponse(c, "Order service is healthy", map[string]interface{}{
		"service": "order-service",
		"status":  "healthy",
	})
}

// StartConsuming listens Rabbitmq events
func (h *OrderHandler) StartConsuming(consumer *messaging.Consumer) error {
	// Order service will listen below rabbitmq events
	routingKeys := []string{
		"saga.saga-orchestrator.order.completed", // Saga completed successfully
		"saga.saga-orchestrator.order.cancelled", // Saga rollback
	}

	return consumer.ConsumeEvents(routingKeys, h.HandleSagaEvent)
}

// HandleSagaEvent process saga events received from Rabbitmq
func (h *OrderHandler) HandleSagaEvent(event events.SagaEvent) error {
	log.Printf("Order service saga event received: %s", event.EventType)
	return h.orderService.ProcessSagaCompletionEvent(event)
}
