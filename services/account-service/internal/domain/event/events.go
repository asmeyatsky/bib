package event

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the interface that all domain events must implement.
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	AggregateID() uuid.UUID
	AggregateType() string
	OccurredAt() time.Time
}

// BaseEvent contains the common fields for all domain events.
type BaseEvent struct {
	ID            uuid.UUID `json:"event_id"`
	Type          string    `json:"event_type"`
	AggregateIDV  uuid.UUID `json:"aggregate_id"`
	AggregateTypeV string   `json:"aggregate_type"`
	Timestamp     time.Time `json:"occurred_at"`
}

func (e BaseEvent) EventID() uuid.UUID      { return e.ID }
func (e BaseEvent) EventType() string        { return e.Type }
func (e BaseEvent) AggregateID() uuid.UUID   { return e.AggregateIDV }
func (e BaseEvent) AggregateType() string    { return e.AggregateTypeV }
func (e BaseEvent) OccurredAt() time.Time    { return e.Timestamp }

func newBaseEvent(eventType string, aggregateID uuid.UUID) BaseEvent {
	return BaseEvent{
		ID:             uuid.New(),
		Type:           eventType,
		AggregateIDV:   aggregateID,
		AggregateTypeV: "CustomerAccount",
		Timestamp:      time.Now(),
	}
}

// AccountOpened is emitted when a new customer account is created.
type AccountOpened struct {
	BaseEvent
	TenantID      uuid.UUID `json:"tenant_id"`
	AccountNumber string    `json:"account_number"`
	AccountType   string    `json:"account_type"`
	Currency      string    `json:"currency"`
	HolderName    string    `json:"holder_name"`
	HolderEmail   string    `json:"holder_email"`
}

// NewAccountOpened creates a new AccountOpened event.
func NewAccountOpened(
	accountID uuid.UUID,
	tenantID uuid.UUID,
	accountNumber string,
	accountType string,
	currency string,
	holderName string,
	holderEmail string,
) AccountOpened {
	return AccountOpened{
		BaseEvent:     newBaseEvent("account.opened", accountID),
		TenantID:      tenantID,
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Currency:      currency,
		HolderName:    holderName,
		HolderEmail:   holderEmail,
	}
}

// AccountActivated is emitted when an account transitions to ACTIVE status.
type AccountActivated struct {
	BaseEvent
	TenantID      uuid.UUID `json:"tenant_id"`
	AccountNumber string    `json:"account_number"`
	ActivatedAt   time.Time `json:"activated_at"`
}

// NewAccountActivated creates a new AccountActivated event.
func NewAccountActivated(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, activatedAt time.Time) AccountActivated {
	return AccountActivated{
		BaseEvent:     newBaseEvent("account.activated", accountID),
		TenantID:      tenantID,
		AccountNumber: accountNumber,
		ActivatedAt:   activatedAt,
	}
}

// AccountFrozen is emitted when an account is frozen.
type AccountFrozen struct {
	BaseEvent
	TenantID      uuid.UUID `json:"tenant_id"`
	AccountNumber string    `json:"account_number"`
	Reason        string    `json:"reason"`
	FrozenAt      time.Time `json:"frozen_at"`
}

// NewAccountFrozen creates a new AccountFrozen event.
func NewAccountFrozen(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, reason string, frozenAt time.Time) AccountFrozen {
	return AccountFrozen{
		BaseEvent:     newBaseEvent("account.frozen", accountID),
		TenantID:      tenantID,
		AccountNumber: accountNumber,
		Reason:        reason,
		FrozenAt:      frozenAt,
	}
}

// AccountUnfrozen is emitted when a frozen account is unfrozen.
type AccountUnfrozen struct {
	BaseEvent
	TenantID      uuid.UUID `json:"tenant_id"`
	AccountNumber string    `json:"account_number"`
	UnfrozenAt    time.Time `json:"unfrozen_at"`
}

// NewAccountUnfrozen creates a new AccountUnfrozen event.
func NewAccountUnfrozen(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, unfrozenAt time.Time) AccountUnfrozen {
	return AccountUnfrozen{
		BaseEvent:     newBaseEvent("account.unfrozen", accountID),
		TenantID:      tenantID,
		AccountNumber: accountNumber,
		UnfrozenAt:    unfrozenAt,
	}
}

// AccountClosed is emitted when an account is closed.
type AccountClosed struct {
	BaseEvent
	TenantID      uuid.UUID `json:"tenant_id"`
	AccountNumber string    `json:"account_number"`
	Reason        string    `json:"reason"`
	ClosedAt      time.Time `json:"closed_at"`
}

// NewAccountClosed creates a new AccountClosed event.
func NewAccountClosed(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, reason string, closedAt time.Time) AccountClosed {
	return AccountClosed{
		BaseEvent:     newBaseEvent("account.closed", accountID),
		TenantID:      tenantID,
		AccountNumber: accountNumber,
		Reason:        reason,
		ClosedAt:      closedAt,
	}
}
