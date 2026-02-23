package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// --- Mocks ---

type mockFraudClient struct {
	err      error
	approved bool
}

func (m *mockFraudClient) AssessTransaction(_ context.Context, _, _ uuid.UUID, _ decimal.Decimal, _ string) (bool, error) {
	return m.approved, m.err
}

type mockRailAdapter struct {
	submitErr error
	statusErr error
	status    valueobject.PaymentStatus
	statusMsg string
}

func (m *mockRailAdapter) Submit(_ context.Context, _ model.PaymentOrder) error {
	return m.submitErr
}

func (m *mockRailAdapter) GetStatus(_ context.Context, _ uuid.UUID) (valueobject.PaymentStatus, string, error) {
	return m.status, m.statusMsg, m.statusErr
}

type mockEventPublisher struct {
	published []events.DomainEvent
}

func (m *mockEventPublisher) Publish(_ context.Context, _ string, evts ...events.DomainEvent) error {
	m.published = append(m.published, evts...)
	return nil
}

// --- Helper ---

func newTestOrder(t *testing.T) model.PaymentOrder {
	t.Helper()
	routing, err := valueobject.NewRoutingInfo("021000021", "123456789")
	require.NoError(t, err)

	order, err := model.NewPaymentOrder(
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		decimal.NewFromInt(10000),
		"USD",
		valueobject.RailACH,
		routing,
		"REF-001",
		"Test payment",
	)
	require.NoError(t, err)
	return order
}

// --- Tests ---

func TestPaymentSaga_SuccessfulFlow(t *testing.T) {
	fraud := &mockFraudClient{approved: true}
	rail := &mockRailAdapter{}
	pub := &mockEventPublisher{}

	orchestrator := NewPaymentSagaOrchestrator(fraud, rail, pub)
	order := newTestOrder(t)

	state, err := orchestrator.Execute(context.Background(), order)
	require.NoError(t, err)

	assert.Equal(t, order.ID(), state.OrderID)
	assert.Equal(t, SagaStepComplete, state.CurrentStep)
	assert.Nil(t, state.FailedStep)
	assert.Empty(t, state.FailureReason)
	assert.NotNil(t, state.CompletedAt)

	// All steps should be completed
	assert.Contains(t, state.CompletedSteps, SagaStepFraudCheck)
	assert.Contains(t, state.CompletedSteps, SagaStepSubmitToRail)
	assert.Contains(t, state.CompletedSteps, SagaStepComplete)
}

func TestPaymentSaga_FraudDecline(t *testing.T) {
	fraud := &mockFraudClient{approved: false}
	rail := &mockRailAdapter{}
	pub := &mockEventPublisher{}

	orchestrator := NewPaymentSagaOrchestrator(fraud, rail, pub)
	order := newTestOrder(t)

	state, err := orchestrator.Execute(context.Background(), order)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fraud check declined")

	assert.Equal(t, order.ID(), state.OrderID)
	require.NotNil(t, state.FailedStep)
	assert.Equal(t, SagaStepFraudCheck, *state.FailedStep)
	assert.Contains(t, state.FailureReason, "declined by fraud check")
	assert.Nil(t, state.CompletedAt)

	// Fraud check should NOT be in completed steps (it failed)
	assert.NotContains(t, state.CompletedSteps, SagaStepFraudCheck)
	// Rail submission should not have been attempted
	assert.NotContains(t, state.CompletedSteps, SagaStepSubmitToRail)
}

func TestPaymentSaga_FraudCheckError(t *testing.T) {
	fraud := &mockFraudClient{err: fmt.Errorf("fraud service unavailable")}
	rail := &mockRailAdapter{}
	pub := &mockEventPublisher{}

	orchestrator := NewPaymentSagaOrchestrator(fraud, rail, pub)
	order := newTestOrder(t)

	state, err := orchestrator.Execute(context.Background(), order)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fraud service unavailable")

	require.NotNil(t, state.FailedStep)
	assert.Equal(t, SagaStepFraudCheck, *state.FailedStep)
	assert.Contains(t, state.FailureReason, "fraud check error")
}

func TestPaymentSaga_RailFailure(t *testing.T) {
	fraud := &mockFraudClient{approved: true}
	rail := &mockRailAdapter{submitErr: fmt.Errorf("ACH processor timeout")}
	pub := &mockEventPublisher{}

	orchestrator := NewPaymentSagaOrchestrator(fraud, rail, pub)
	order := newTestOrder(t)

	state, err := orchestrator.Execute(context.Background(), order)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ACH processor timeout")

	// Fraud check should have passed
	assert.Contains(t, state.CompletedSteps, SagaStepFraudCheck)

	// Rail submission should have failed
	require.NotNil(t, state.FailedStep)
	assert.Equal(t, SagaStepSubmitToRail, *state.FailedStep)
	assert.Contains(t, state.FailureReason, "rail submission error")
	assert.Nil(t, state.CompletedAt)
}

func TestPaymentSaga_NilFraudClient_SkipsFraudCheck(t *testing.T) {
	rail := &mockRailAdapter{}
	pub := &mockEventPublisher{}

	// Pass nil fraud client
	orchestrator := NewPaymentSagaOrchestrator(nil, rail, pub)
	order := newTestOrder(t)

	state, err := orchestrator.Execute(context.Background(), order)
	require.NoError(t, err)

	// Should still complete successfully, skipping fraud check
	assert.Equal(t, SagaStepComplete, state.CurrentStep)
	assert.NotNil(t, state.CompletedAt)
	assert.Contains(t, state.CompletedSteps, SagaStepFraudCheck)
	assert.Contains(t, state.CompletedSteps, SagaStepSubmitToRail)
	assert.Contains(t, state.CompletedSteps, SagaStepComplete)
}
