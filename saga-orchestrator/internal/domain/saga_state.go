package domain

import (
	"github.com/google/uuid"
	"time"
)

type SagaStatus string

const (
	SagaStatusStarted      SagaStatus = "started"
	SagaStatusInProgress   SagaStatus = "in_progress"
	SagaStatusCompleted    SagaStatus = "completed"
	SagaStatusFailed       SagaStatus = "failed"
	SagaStatusCompensating SagaStatus = "compensating"
	SagaStatusCompensated  SagaStatus = "compensated"
)

type SagaStep string

const (
	// Forward steps
	StepOrderCreated      SagaStep = "order_created"
	StepPaymentProcessed  SagaStep = "payment_processed"
	StepInventoryReserved SagaStep = "inventory_reserved"
	StepShippingCreated   SagaStep = "shipping_created"
	StepNotificationSent  SagaStep = "notification_sent"

	// Compensation steps
	StepOrderCancelled    SagaStep = "order_cancelled"
	StepPaymentRefunded   SagaStep = "payment_refunded"
	StepInventoryReleased SagaStep = "inventory_released"
	StepShippingCancelled SagaStep = "shipping_cancelled"
)

type SagaInstance struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	OrderID          uuid.UUID  `json:"order_id" db:"order_id"`
	CustomerID       uuid.UUID  `json:"customer_id" db:"customer_id"`
	Status           SagaStatus `json:"status" db:"status"`
	CurrentStep      SagaStep   `json:"current_step" db:"current_step"`
	CompletedSteps   []SagaStep `json:"completed_steps" db:"completed_steps"`
	FailureReason    string     `json:"failure_reason,omitempty" db:"failure_reason"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CompensatedSteps []SagaStep `json:"compensated_steps" db:"compensated_steps"`

	// All data during saga
	Context map[string]interface{} `json:"context" db:"context"`
}

func (s *SagaInstance) IsStepCompleted(step SagaStep) bool {
	for _, completedStep := range s.CompletedSteps {
		if completedStep == step {
			return true
		}
	}
	return false
}

func (s *SagaInstance) MarkStepCompleted(step SagaStep) {
	if !s.IsStepCompleted(step) {
		s.CompletedSteps = append(s.CompletedSteps, step)
	}
	s.CurrentStep = step
	s.UpdatedAt = time.Now()
}

func (s *SagaInstance) GetNextStep() SagaStep {
	switch s.CurrentStep {
	case StepOrderCreated:
		return StepPaymentProcessed
	case StepPaymentProcessed:
		return StepInventoryReserved
	case StepInventoryReserved:
		return StepShippingCreated
	case StepShippingCreated:
		return StepNotificationSent
	default:
		return "" // Last step
	}
}

func (s *SagaInstance) GetCompensationStep() SagaStep {
	// Shipping completed but compensation not worked
	if s.IsStepCompleted(StepShippingCreated) && !s.IsCompensationCompleted(StepShippingCancelled) {
		return StepShippingCancelled
	}
	if s.IsStepCompleted(StepInventoryReserved) && !s.IsCompensationCompleted(StepInventoryReleased) {
		return StepInventoryReleased
	}
	if s.IsStepCompleted(StepPaymentProcessed) && !s.IsCompensationCompleted(StepPaymentRefunded) {
		return StepPaymentRefunded
	}
	if s.IsStepCompleted(StepOrderCreated) && !s.IsCompensationCompleted(StepOrderCancelled) {
		return StepOrderCancelled
	}
	return ""
}

func (s *SagaInstance) IsCompensationCompleted(compensationStep SagaStep) bool {
	for _, compensated := range s.CompensatedSteps {
		if compensated == compensationStep {
			return true
		}
	}
	return false
}

func (s *SagaInstance) MarkCompensationCompleted(compensationStep SagaStep) {
	if !s.IsCompensationCompleted(compensationStep) {
		s.CompensatedSteps = append(s.CompensatedSteps, compensationStep)
	}
	s.UpdatedAt = time.Now()
}
