package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/google/uuid"
)

// SagaStep represents a step in the payment saga.
type SagaStep string

const (
	SagaStepFraudCheck   SagaStep = "FRAUD_CHECK"
	SagaStepReserveFunds SagaStep = "RESERVE_FUNDS"
	SagaStepSubmitToRail SagaStep = "SUBMIT_TO_RAIL"
	SagaStepPostToLedger SagaStep = "POST_TO_LEDGER"
	SagaStepComplete     SagaStep = "COMPLETE"
)

// SagaState tracks the current state of a payment saga.
type SagaState struct {
	StartedAt      time.Time
	FailedStep     *SagaStep
	CompletedAt    *time.Time
	CurrentStep    SagaStep
	FailureReason  string
	CompletedSteps []SagaStep
	OrderID        uuid.UUID
}

// PaymentSagaOrchestrator manages the payment saga workflow.
type PaymentSagaOrchestrator struct {
	fraudClient port.FraudClient
	railAdapter port.RailAdapter
	publisher   port.EventPublisher
}

func NewPaymentSagaOrchestrator(fraudClient port.FraudClient, railAdapter port.RailAdapter, publisher port.EventPublisher) *PaymentSagaOrchestrator {
	return &PaymentSagaOrchestrator{
		fraudClient: fraudClient,
		railAdapter: railAdapter,
		publisher:   publisher,
	}
}

// Execute runs the payment saga for the given order.
func (o *PaymentSagaOrchestrator) Execute(ctx context.Context, order model.PaymentOrder) (SagaState, error) {
	state := SagaState{
		OrderID:     order.ID(),
		CurrentStep: SagaStepFraudCheck,
		StartedAt:   time.Now().UTC(),
	}

	// Step 1: Fraud check
	if o.fraudClient != nil {
		approved, err := o.fraudClient.AssessTransaction(ctx, order.TenantID(), order.SourceAccountID(), order.Amount(), order.Currency())
		if err != nil {
			return o.failSaga(state, SagaStepFraudCheck, fmt.Sprintf("fraud check error: %v", err)), err
		}
		if !approved {
			return o.failSaga(state, SagaStepFraudCheck, "transaction declined by fraud check"), fmt.Errorf("fraud check declined")
		}
	}
	state.CompletedSteps = append(state.CompletedSteps, SagaStepFraudCheck)

	// Step 2: Submit to rail
	state.CurrentStep = SagaStepSubmitToRail
	if err := o.railAdapter.Submit(ctx, order); err != nil {
		return o.failSaga(state, SagaStepSubmitToRail, fmt.Sprintf("rail submission error: %v", err)), err
	}
	state.CompletedSteps = append(state.CompletedSteps, SagaStepSubmitToRail)

	// Step 3: Complete
	state.CurrentStep = SagaStepComplete
	state.CompletedSteps = append(state.CompletedSteps, SagaStepComplete)
	now := time.Now().UTC()
	state.CompletedAt = &now

	return state, nil
}

func (o *PaymentSagaOrchestrator) failSaga(state SagaState, step SagaStep, reason string) SagaState {
	state.FailedStep = &step
	state.FailureReason = reason
	return state
}
