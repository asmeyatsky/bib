package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypePaymentOrder = "PaymentOrder"

// PaymentInitiated is emitted when a new payment order is created.
type PaymentInitiated struct {
	events.BaseEvent
	PaymentID uuid.UUID       `json:"payment_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency"`
	Rail      string          `json:"rail"`
}

func NewPaymentInitiated(paymentID, tenantID uuid.UUID, amount decimal.Decimal, currency, rail string) PaymentInitiated {
	payload, _ := json.Marshal(struct {
		PaymentID uuid.UUID       `json:"payment_id"`
		TenantID  uuid.UUID       `json:"tenant_id"`
		Amount    decimal.Decimal `json:"amount"`
		Currency  string          `json:"currency"`
		Rail      string          `json:"rail"`
	}{paymentID, tenantID, amount, currency, rail})

	return PaymentInitiated{
		BaseEvent: events.NewBaseEvent("payment.order.initiated", paymentID, AggregateTypePaymentOrder, payload),
		PaymentID: paymentID,
		TenantID:  tenantID,
		Amount:    amount,
		Currency:  currency,
		Rail:      rail,
	}
}

// PaymentProcessing is emitted when a payment order begins processing via a rail adapter.
type PaymentProcessing struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Rail      string    `json:"rail"`
}

func NewPaymentProcessing(paymentID, tenantID uuid.UUID, rail string) PaymentProcessing {
	payload, _ := json.Marshal(struct {
		PaymentID uuid.UUID `json:"payment_id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		Rail      string    `json:"rail"`
	}{paymentID, tenantID, rail})

	return PaymentProcessing{
		BaseEvent: events.NewBaseEvent("payment.order.processing", paymentID, AggregateTypePaymentOrder, payload),
		PaymentID: paymentID,
		TenantID:  tenantID,
		Rail:      rail,
	}
}

// PaymentSettled is emitted when a payment order is successfully settled.
type PaymentSettled struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	SettledAt time.Time `json:"settled_at"`
}

func NewPaymentSettled(paymentID, tenantID uuid.UUID, settledAt time.Time) PaymentSettled {
	payload, _ := json.Marshal(struct {
		PaymentID uuid.UUID `json:"payment_id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		SettledAt time.Time `json:"settled_at"`
	}{paymentID, tenantID, settledAt})

	return PaymentSettled{
		BaseEvent: events.NewBaseEvent("payment.order.settled", paymentID, AggregateTypePaymentOrder, payload),
		PaymentID: paymentID,
		TenantID:  tenantID,
		SettledAt: settledAt,
	}
}

// PaymentFailed is emitted when a payment order fails.
type PaymentFailed struct {
	events.BaseEvent
	PaymentID     uuid.UUID `json:"payment_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	FailureReason string    `json:"failure_reason"`
}

func NewPaymentFailed(paymentID, tenantID uuid.UUID, reason string) PaymentFailed {
	payload, _ := json.Marshal(struct {
		PaymentID     uuid.UUID `json:"payment_id"`
		TenantID      uuid.UUID `json:"tenant_id"`
		FailureReason string    `json:"failure_reason"`
	}{paymentID, tenantID, reason})

	return PaymentFailed{
		BaseEvent:     events.NewBaseEvent("payment.order.failed", paymentID, AggregateTypePaymentOrder, payload),
		PaymentID:     paymentID,
		TenantID:      tenantID,
		FailureReason: reason,
	}
}

// PaymentReversed is emitted when a settled payment order is reversed.
type PaymentReversed struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Reason    string    `json:"reason"`
}

func NewPaymentReversed(paymentID, tenantID uuid.UUID, reason string) PaymentReversed {
	payload, _ := json.Marshal(struct {
		PaymentID uuid.UUID `json:"payment_id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		Reason    string    `json:"reason"`
	}{paymentID, tenantID, reason})

	return PaymentReversed{
		BaseEvent: events.NewBaseEvent("payment.order.reversed", paymentID, AggregateTypePaymentOrder, payload),
		PaymentID: paymentID,
		TenantID:  tenantID,
		Reason:    reason,
	}
}
