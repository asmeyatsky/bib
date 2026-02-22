package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

func newTestPaymentOrder(t *testing.T) model.PaymentOrder {
	t.Helper()
	routingInfo, err := valueobject.NewRoutingInfo("021000021", "123456789")
	require.NoError(t, err)

	order, err := model.NewPaymentOrder(
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		decimal.NewFromInt(1000),
		"USD",
		valueobject.RailACH,
		routingInfo,
		"REF-001",
		"Test payment",
	)
	require.NoError(t, err)
	return order
}

func TestNewPaymentOrder_Valid(t *testing.T) {
	tenantID := uuid.New()
	sourceAcctID := uuid.New()
	routingInfo, err := valueobject.NewRoutingInfo("021000021", "123456789")
	require.NoError(t, err)

	order, err := model.NewPaymentOrder(
		tenantID,
		sourceAcctID,
		uuid.Nil,
		decimal.NewFromInt(500),
		"USD",
		valueobject.RailACH,
		routingInfo,
		"REF-001",
		"Payment for invoice",
	)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, order.ID())
	assert.Equal(t, tenantID, order.TenantID())
	assert.Equal(t, sourceAcctID, order.SourceAccountID())
	assert.Equal(t, uuid.Nil, order.DestinationAccountID())
	assert.True(t, decimal.NewFromInt(500).Equal(order.Amount()))
	assert.Equal(t, "USD", order.Currency())
	assert.Equal(t, valueobject.RailACH, order.Rail())
	assert.Equal(t, valueobject.PaymentStatusInitiated, order.Status())
	assert.Equal(t, "021000021", order.RoutingInfo().RoutingNumber())
	assert.Equal(t, "123456789", order.RoutingInfo().ExternalAccountNumber())
	assert.Equal(t, "REF-001", order.Reference())
	assert.Equal(t, "Payment for invoice", order.Description())
	assert.Equal(t, "", order.FailureReason())
	assert.Nil(t, order.SettledAt())
	assert.Equal(t, 1, order.Version())
	assert.False(t, order.CreatedAt().IsZero())
	assert.False(t, order.UpdatedAt().IsZero())
	assert.False(t, order.InitiatedAt().IsZero())

	// NewPaymentOrder should emit a PaymentInitiated event.
	events := order.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "payment.order.initiated", events[0].EventType())
	assert.Equal(t, order.ID().String(), events[0].AggregateID())
}

func TestNewPaymentOrder_InternalTransfer(t *testing.T) {
	destAcctID := uuid.New()
	routingInfo, _ := valueobject.NewRoutingInfo("", "")

	order, err := model.NewPaymentOrder(
		uuid.New(),
		uuid.New(),
		destAcctID,
		decimal.NewFromInt(250),
		"USD",
		valueobject.RailInternal,
		routingInfo,
		"INTERNAL-001",
		"Internal transfer",
	)
	require.NoError(t, err)

	assert.Equal(t, destAcctID, order.DestinationAccountID())
	assert.Equal(t, valueobject.RailInternal, order.Rail())
}

func TestNewPaymentOrder_MissingTenantID(t *testing.T) {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	_, err := model.NewPaymentOrder(
		uuid.Nil,
		uuid.New(),
		uuid.Nil,
		decimal.NewFromInt(100),
		"USD",
		valueobject.RailACH,
		routingInfo,
		"REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewPaymentOrder_MissingSourceAccount(t *testing.T) {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	_, err := model.NewPaymentOrder(
		uuid.New(),
		uuid.Nil,
		uuid.Nil,
		decimal.NewFromInt(100),
		"USD",
		valueobject.RailACH,
		routingInfo,
		"REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source account ID is required")
}

func TestNewPaymentOrder_NonPositiveAmount(t *testing.T) {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")

	_, err := model.NewPaymentOrder(
		uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(0), "USD", valueobject.RailACH,
		routingInfo, "REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	_, err = model.NewPaymentOrder(
		uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(-100), "USD", valueobject.RailACH,
		routingInfo, "REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestNewPaymentOrder_MissingCurrency(t *testing.T) {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	_, err := model.NewPaymentOrder(
		uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(100), "", valueobject.RailACH,
		routingInfo, "REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency is required")
}

func TestNewPaymentOrder_MissingRail(t *testing.T) {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	_, err := model.NewPaymentOrder(
		uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(100), "USD", valueobject.PaymentRail{},
		routingInfo, "REF", "desc",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment rail is required")
}

// --- Full lifecycle tests ---

func TestPaymentOrder_Lifecycle_InitiateProcessSettle(t *testing.T) {
	order := newTestPaymentOrder(t)
	assert.Equal(t, valueobject.PaymentStatusInitiated, order.Status())

	// Transition to PROCESSING.
	now := time.Now().UTC()
	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusProcessing, processing.Status())
	assert.Equal(t, 2, processing.Version())
	assert.Equal(t, now, processing.UpdatedAt())

	// Original remains immutable.
	assert.Equal(t, valueobject.PaymentStatusInitiated, order.Status())
	assert.Equal(t, 1, order.Version())

	// Transition to SETTLED.
	settleTime := now.Add(time.Hour)
	settled, err := processing.Settle(settleTime)
	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusSettled, settled.Status())
	assert.Equal(t, 3, settled.Version())
	assert.NotNil(t, settled.SettledAt())
	assert.Equal(t, settleTime, *settled.SettledAt())

	// Processing remains immutable.
	assert.Equal(t, valueobject.PaymentStatusProcessing, processing.Status())
	assert.Nil(t, processing.SettledAt())

	// Events should contain: initiated, processing, settled.
	events := settled.DomainEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "payment.order.initiated", events[0].EventType())
	assert.Equal(t, "payment.order.processing", events[1].EventType())
	assert.Equal(t, "payment.order.settled", events[2].EventType())
}

func TestPaymentOrder_Lifecycle_InitiateProcessFail(t *testing.T) {
	order := newTestPaymentOrder(t)

	now := time.Now().UTC()
	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)

	failTime := now.Add(time.Minute)
	failed, err := processing.Fail("insufficient funds at clearing house", failTime)
	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusFailed, failed.Status())
	assert.Equal(t, 3, failed.Version())
	assert.Equal(t, "insufficient funds at clearing house", failed.FailureReason())
	assert.True(t, failed.Status().IsTerminal())

	events := failed.DomainEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "payment.order.initiated", events[0].EventType())
	assert.Equal(t, "payment.order.processing", events[1].EventType())
	assert.Equal(t, "payment.order.failed", events[2].EventType())
}

func TestPaymentOrder_Lifecycle_SettleThenReverse(t *testing.T) {
	order := newTestPaymentOrder(t)

	now := time.Now().UTC()
	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)

	settleTime := now.Add(time.Hour)
	settled, err := processing.Settle(settleTime)
	require.NoError(t, err)

	reverseTime := settleTime.Add(24 * time.Hour)
	reversed, err := settled.Reverse("customer dispute", reverseTime)
	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusReversed, reversed.Status())
	assert.Equal(t, 4, reversed.Version())
	assert.Equal(t, "customer dispute", reversed.FailureReason())
	assert.True(t, reversed.Status().IsTerminal())

	events := reversed.DomainEvents()
	require.Len(t, events, 4)
	assert.Equal(t, "payment.order.initiated", events[0].EventType())
	assert.Equal(t, "payment.order.processing", events[1].EventType())
	assert.Equal(t, "payment.order.settled", events[2].EventType())
	assert.Equal(t, "payment.order.reversed", events[3].EventType())
}

// --- Invalid transition tests ---

func TestPaymentOrder_MarkProcessing_NotFromInitiated_Error(t *testing.T) {
	order := newTestPaymentOrder(t)
	now := time.Now().UTC()

	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)

	// Cannot mark processing again.
	_, err = processing.MarkProcessing(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only mark processing from INITIATED status")
}

func TestPaymentOrder_Settle_NotFromProcessing_Error(t *testing.T) {
	order := newTestPaymentOrder(t)
	now := time.Now().UTC()

	// Cannot settle from INITIATED.
	_, err := order.Settle(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only settle from PROCESSING status")
}

func TestPaymentOrder_Fail_NotFromProcessing_Error(t *testing.T) {
	order := newTestPaymentOrder(t)
	now := time.Now().UTC()

	// Cannot fail from INITIATED.
	_, err := order.Fail("reason", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only fail from PROCESSING status")
}

func TestPaymentOrder_Reverse_NotFromSettled_Error(t *testing.T) {
	order := newTestPaymentOrder(t)
	now := time.Now().UTC()

	// Cannot reverse from INITIATED.
	_, err := order.Reverse("reason", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse from SETTLED status")

	// Cannot reverse from PROCESSING.
	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)
	_, err = processing.Reverse("reason", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse from SETTLED status")

	// Cannot reverse from FAILED.
	failed, err := processing.Fail("fail reason", now)
	require.NoError(t, err)
	_, err = failed.Reverse("reason", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse from SETTLED status")
}

func TestPaymentOrder_DoubleReverse_Error(t *testing.T) {
	order := newTestPaymentOrder(t)
	now := time.Now().UTC()

	processing, err := order.MarkProcessing(now)
	require.NoError(t, err)

	settled, err := processing.Settle(now)
	require.NoError(t, err)

	reversed, err := settled.Reverse("first reversal", now)
	require.NoError(t, err)

	_, err = reversed.Reverse("second reversal", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse from SETTLED status")
}

func TestPaymentOrder_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	sourceAcctID := uuid.New()
	destAcctID := uuid.New()
	amount := decimal.NewFromInt(999)
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "987654321")
	settledAt := time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, time.March, 14, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC)
	initiatedAt := time.Date(2024, time.March, 14, 10, 0, 0, 0, time.UTC)

	order := model.Reconstruct(
		id, tenantID, sourceAcctID, destAcctID,
		amount, "EUR", valueobject.RailSEPA, valueobject.PaymentStatusSettled,
		routingInfo, "REF-R", "Reconstructed payment", "",
		initiatedAt, &settledAt, 3, createdAt, updatedAt,
	)

	assert.Equal(t, id, order.ID())
	assert.Equal(t, tenantID, order.TenantID())
	assert.Equal(t, sourceAcctID, order.SourceAccountID())
	assert.Equal(t, destAcctID, order.DestinationAccountID())
	assert.True(t, amount.Equal(order.Amount()))
	assert.Equal(t, "EUR", order.Currency())
	assert.Equal(t, valueobject.RailSEPA, order.Rail())
	assert.Equal(t, valueobject.PaymentStatusSettled, order.Status())
	assert.Equal(t, "021000021", order.RoutingInfo().RoutingNumber())
	assert.Equal(t, "REF-R", order.Reference())
	assert.Equal(t, "Reconstructed payment", order.Description())
	assert.Equal(t, 3, order.Version())
	assert.Equal(t, createdAt, order.CreatedAt())
	assert.Equal(t, updatedAt, order.UpdatedAt())
	assert.NotNil(t, order.SettledAt())
	assert.Equal(t, settledAt, *order.SettledAt())
	assert.Empty(t, order.DomainEvents())
}

func TestPaymentOrder_ClearDomainEvents(t *testing.T) {
	order := newTestPaymentOrder(t)
	require.Len(t, order.DomainEvents(), 1)

	cleared, updated := order.ClearDomainEvents()
	assert.Len(t, cleared, 1)
	assert.Equal(t, "payment.order.initiated", cleared[0].EventType())
	assert.Empty(t, updated.DomainEvents())
}

func TestPaymentOrder_Immutability_MarkProcessingDoesNotMutateOriginal(t *testing.T) {
	order := newTestPaymentOrder(t)
	originalVersion := order.Version()
	originalStatus := order.Status()

	now := time.Now().UTC()
	_, err := order.MarkProcessing(now)
	require.NoError(t, err)

	assert.Equal(t, originalVersion, order.Version())
	assert.Equal(t, originalStatus, order.Status())
}
