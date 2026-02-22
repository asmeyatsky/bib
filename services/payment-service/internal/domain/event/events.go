package event

import (
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
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency"`
	Rail      string          `json:"rail"`
}

func NewPaymentInitiated(paymentID, tenantID uuid.UUID, amount decimal.Decimal, currency, rail string) PaymentInitiated {
	return PaymentInitiated{
		BaseEvent: events.NewBaseEvent("payment.order.initiated", paymentID.String(), AggregateTypePaymentOrder, tenantID.String()),
		PaymentID: paymentID,
		Amount:    amount,
		Currency:  currency,
		Rail:      rail,
	}
}

// PaymentProcessing is emitted when a payment order begins processing via a rail adapter.
type PaymentProcessing struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	Rail      string    `json:"rail"`
}

func NewPaymentProcessing(paymentID, tenantID uuid.UUID, rail string) PaymentProcessing {
	return PaymentProcessing{
		BaseEvent: events.NewBaseEvent("payment.order.processing", paymentID.String(), AggregateTypePaymentOrder, tenantID.String()),
		PaymentID: paymentID,
		Rail:      rail,
	}
}

// PaymentSettled is emitted when a payment order is successfully settled.
type PaymentSettled struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	SettledAt time.Time `json:"settled_at"`
}

func NewPaymentSettled(paymentID, tenantID uuid.UUID, settledAt time.Time) PaymentSettled {
	return PaymentSettled{
		BaseEvent: events.NewBaseEvent("payment.order.settled", paymentID.String(), AggregateTypePaymentOrder, tenantID.String()),
		PaymentID: paymentID,
		SettledAt: settledAt,
	}
}

// PaymentFailed is emitted when a payment order fails.
type PaymentFailed struct {
	events.BaseEvent
	PaymentID     uuid.UUID `json:"payment_id"`
	FailureReason string    `json:"failure_reason"`
}

func NewPaymentFailed(paymentID, tenantID uuid.UUID, reason string) PaymentFailed {
	return PaymentFailed{
		BaseEvent:     events.NewBaseEvent("payment.order.failed", paymentID.String(), AggregateTypePaymentOrder, tenantID.String()),
		PaymentID:     paymentID,
		FailureReason: reason,
	}
}

// PaymentReversed is emitted when a settled payment order is reversed.
type PaymentReversed struct {
	events.BaseEvent
	PaymentID uuid.UUID `json:"payment_id"`
	Reason    string    `json:"reason"`
}

func NewPaymentReversed(paymentID, tenantID uuid.UUID, reason string) PaymentReversed {
	return PaymentReversed{
		BaseEvent: events.NewBaseEvent("payment.order.reversed", paymentID.String(), AggregateTypePaymentOrder, tenantID.String()),
		PaymentID: paymentID,
		Reason:    reason,
	}
}
