package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/payment-service/internal/domain/event"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// PaymentOrder is the root aggregate for the payment bounded context.
// It represents an immutable payment instruction moving through its lifecycle.
type PaymentOrder struct {
	id                   uuid.UUID
	tenantID             uuid.UUID
	sourceAccountID      uuid.UUID
	destinationAccountID uuid.UUID // internal account, or uuid.Nil for external
	amount               decimal.Decimal
	currency             string
	rail                 valueobject.PaymentRail
	status               valueobject.PaymentStatus
	routingInfo          valueobject.RoutingInfo
	reference            string
	description          string
	failureReason        string
	initiatedAt          time.Time
	settledAt            *time.Time
	version              int
	createdAt            time.Time
	updatedAt            time.Time
	domainEvents         []events.DomainEvent
}

// NewPaymentOrder creates a new payment order in INITIATED status.
func NewPaymentOrder(
	tenantID uuid.UUID,
	sourceAccountID uuid.UUID,
	destinationAccountID uuid.UUID,
	amount decimal.Decimal,
	currency string,
	rail valueobject.PaymentRail,
	routingInfo valueobject.RoutingInfo,
	reference string,
	description string,
) (PaymentOrder, error) {
	if tenantID == uuid.Nil {
		return PaymentOrder{}, fmt.Errorf("tenant ID is required")
	}
	if sourceAccountID == uuid.Nil {
		return PaymentOrder{}, fmt.Errorf("source account ID is required")
	}
	if !amount.IsPositive() {
		return PaymentOrder{}, fmt.Errorf("amount must be positive, got: %s", amount.String())
	}
	if currency == "" {
		return PaymentOrder{}, fmt.Errorf("currency is required")
	}
	if rail.IsZero() {
		return PaymentOrder{}, fmt.Errorf("payment rail is required")
	}

	now := time.Now().UTC()
	id := uuid.New()

	order := PaymentOrder{
		id:                   id,
		tenantID:             tenantID,
		sourceAccountID:      sourceAccountID,
		destinationAccountID: destinationAccountID,
		amount:               amount,
		currency:             currency,
		rail:                 rail,
		status:               valueobject.PaymentStatusInitiated,
		routingInfo:          routingInfo,
		reference:            reference,
		description:          description,
		initiatedAt:          now,
		version:              1,
		createdAt:            now,
		updatedAt:            now,
	}

	order.domainEvents = append(order.domainEvents,
		event.NewPaymentInitiated(id, tenantID, amount, currency, rail.String()),
	)

	return order, nil
}

// Reconstruct recreates a PaymentOrder from persistence (no validation, no events).
func Reconstruct(
	id, tenantID, sourceAccountID, destinationAccountID uuid.UUID,
	amount decimal.Decimal,
	currency string,
	rail valueobject.PaymentRail,
	status valueobject.PaymentStatus,
	routingInfo valueobject.RoutingInfo,
	reference, description, failureReason string,
	initiatedAt time.Time,
	settledAt *time.Time,
	version int,
	createdAt, updatedAt time.Time,
) PaymentOrder {
	return PaymentOrder{
		id:                   id,
		tenantID:             tenantID,
		sourceAccountID:      sourceAccountID,
		destinationAccountID: destinationAccountID,
		amount:               amount,
		currency:             currency,
		rail:                 rail,
		status:               status,
		routingInfo:          routingInfo,
		reference:            reference,
		description:          description,
		failureReason:        failureReason,
		initiatedAt:          initiatedAt,
		settledAt:            settledAt,
		version:              version,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
	}
}

// MarkProcessing transitions the order from INITIATED to PROCESSING (immutable - returns new copy).
func (po PaymentOrder) MarkProcessing(now time.Time) (PaymentOrder, error) {
	if po.status != valueobject.PaymentStatusInitiated {
		return PaymentOrder{}, fmt.Errorf("can only mark processing from INITIATED status, current: %s", po.status.String())
	}

	updated := po
	updated.status = valueobject.PaymentStatusProcessing
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = append([]events.DomainEvent{}, po.domainEvents...)
	updated.domainEvents = append(updated.domainEvents,
		event.NewPaymentProcessing(po.id, po.tenantID, po.rail.String()),
	)
	return updated, nil
}

// Settle transitions the order from PROCESSING to SETTLED (immutable - returns new copy).
func (po PaymentOrder) Settle(now time.Time) (PaymentOrder, error) {
	if po.status != valueobject.PaymentStatusProcessing {
		return PaymentOrder{}, fmt.Errorf("can only settle from PROCESSING status, current: %s", po.status.String())
	}

	updated := po
	updated.status = valueobject.PaymentStatusSettled
	updated.settledAt = &now
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = append([]events.DomainEvent{}, po.domainEvents...)
	updated.domainEvents = append(updated.domainEvents,
		event.NewPaymentSettled(po.id, po.tenantID, now),
	)
	return updated, nil
}

// Fail transitions the order from PROCESSING to FAILED (immutable - returns new copy).
func (po PaymentOrder) Fail(reason string, now time.Time) (PaymentOrder, error) {
	if po.status != valueobject.PaymentStatusProcessing {
		return PaymentOrder{}, fmt.Errorf("can only fail from PROCESSING status, current: %s", po.status.String())
	}

	updated := po
	updated.status = valueobject.PaymentStatusFailed
	updated.failureReason = reason
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = append([]events.DomainEvent{}, po.domainEvents...)
	updated.domainEvents = append(updated.domainEvents,
		event.NewPaymentFailed(po.id, po.tenantID, reason),
	)
	return updated, nil
}

// Reverse transitions the order from SETTLED to REVERSED (immutable - returns new copy).
func (po PaymentOrder) Reverse(reason string, now time.Time) (PaymentOrder, error) {
	if po.status != valueobject.PaymentStatusSettled {
		return PaymentOrder{}, fmt.Errorf("can only reverse from SETTLED status, current: %s", po.status.String())
	}

	updated := po
	updated.status = valueobject.PaymentStatusReversed
	updated.failureReason = reason
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = append([]events.DomainEvent{}, po.domainEvents...)
	updated.domainEvents = append(updated.domainEvents,
		event.NewPaymentReversed(po.id, po.tenantID, reason),
	)
	return updated, nil
}

// Accessors

func (po PaymentOrder) ID() uuid.UUID                        { return po.id }
func (po PaymentOrder) TenantID() uuid.UUID                  { return po.tenantID }
func (po PaymentOrder) SourceAccountID() uuid.UUID           { return po.sourceAccountID }
func (po PaymentOrder) DestinationAccountID() uuid.UUID      { return po.destinationAccountID }
func (po PaymentOrder) Amount() decimal.Decimal               { return po.amount }
func (po PaymentOrder) Currency() string                      { return po.currency }
func (po PaymentOrder) Rail() valueobject.PaymentRail         { return po.rail }
func (po PaymentOrder) Status() valueobject.PaymentStatus     { return po.status }
func (po PaymentOrder) RoutingInfo() valueobject.RoutingInfo  { return po.routingInfo }
func (po PaymentOrder) Reference() string                     { return po.reference }
func (po PaymentOrder) Description() string                   { return po.description }
func (po PaymentOrder) FailureReason() string                 { return po.failureReason }
func (po PaymentOrder) InitiatedAt() time.Time                { return po.initiatedAt }
func (po PaymentOrder) SettledAt() *time.Time                 { return po.settledAt }
func (po PaymentOrder) Version() int                          { return po.version }
func (po PaymentOrder) CreatedAt() time.Time                  { return po.createdAt }
func (po PaymentOrder) UpdatedAt() time.Time                  { return po.updatedAt }
func (po PaymentOrder) DomainEvents() []events.DomainEvent    { return po.domainEvents }

// ClearDomainEvents returns the collected domain events and a new PaymentOrder with events cleared.
func (po PaymentOrder) ClearDomainEvents() ([]events.DomainEvent, PaymentOrder) {
	evts := po.domainEvents
	po.domainEvents = nil
	return evts, po
}
