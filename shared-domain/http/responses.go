package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func SuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func CreatedResponse(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func BadRequestResponse(c *fiber.Ctx, message string, details map[string]interface{}) error {
	return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    "BAD_REQUEST",
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func NotFoundResponse(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: message,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func InternalServerErrorResponse(c *fiber.Ctx, message string, details map[string]interface{}) error {
	return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func ConflictResponse(c *fiber.Ctx, message string, details map[string]interface{}) error {
	return c.Status(fiber.StatusConflict).JSON(APIResponse{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    "CONFLICT",
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

func getRequestID(c *fiber.Ctx) string {
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		c.Set("X-Request-ID", requestID)
	}
	return requestID
}
