package event

import (
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

// DomainEvent is an alias for the shared pkg/events.DomainEvent interface.
type DomainEvent = events.DomainEvent

// AccountOpened is emitted when a new customer account is created.
type AccountOpened struct {
	events.BaseEvent
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	Currency      string `json:"currency"`
	HolderName    string `json:"holder_name"`
	HolderEmail   string `json:"holder_email"`
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
		BaseEvent:     events.NewBaseEvent("account.opened", accountID.String(), "CustomerAccount", tenantID.String()),
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Currency:      currency,
		HolderName:    holderName,
		HolderEmail:   holderEmail,
	}
}

// AccountActivated is emitted when an account transitions to ACTIVE status.
type AccountActivated struct {
	events.BaseEvent
	AccountNumber string    `json:"account_number"`
	ActivatedAt   time.Time `json:"activated_at"`
}

// NewAccountActivated creates a new AccountActivated event.
func NewAccountActivated(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, activatedAt time.Time) AccountActivated {
	return AccountActivated{
		BaseEvent:     events.NewBaseEvent("account.activated", accountID.String(), "CustomerAccount", tenantID.String()),
		AccountNumber: accountNumber,
		ActivatedAt:   activatedAt,
	}
}

// AccountFrozen is emitted when an account is frozen.
type AccountFrozen struct {
	events.BaseEvent
	AccountNumber string    `json:"account_number"`
	Reason        string    `json:"reason"`
	FrozenAt      time.Time `json:"frozen_at"`
}

// NewAccountFrozen creates a new AccountFrozen event.
func NewAccountFrozen(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, reason string, frozenAt time.Time) AccountFrozen {
	return AccountFrozen{
		BaseEvent:     events.NewBaseEvent("account.frozen", accountID.String(), "CustomerAccount", tenantID.String()),
		AccountNumber: accountNumber,
		Reason:        reason,
		FrozenAt:      frozenAt,
	}
}

// AccountUnfrozen is emitted when a frozen account is unfrozen.
type AccountUnfrozen struct {
	events.BaseEvent
	AccountNumber string    `json:"account_number"`
	UnfrozenAt    time.Time `json:"unfrozen_at"`
}

// NewAccountUnfrozen creates a new AccountUnfrozen event.
func NewAccountUnfrozen(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, unfrozenAt time.Time) AccountUnfrozen {
	return AccountUnfrozen{
		BaseEvent:     events.NewBaseEvent("account.unfrozen", accountID.String(), "CustomerAccount", tenantID.String()),
		AccountNumber: accountNumber,
		UnfrozenAt:    unfrozenAt,
	}
}

// AccountClosed is emitted when an account is closed.
type AccountClosed struct {
	events.BaseEvent
	AccountNumber string    `json:"account_number"`
	Reason        string    `json:"reason"`
	ClosedAt      time.Time `json:"closed_at"`
}

// NewAccountClosed creates a new AccountClosed event.
func NewAccountClosed(accountID uuid.UUID, tenantID uuid.UUID, accountNumber string, reason string, closedAt time.Time) AccountClosed {
	return AccountClosed{
		BaseEvent:     events.NewBaseEvent("account.closed", accountID.String(), "CustomerAccount", tenantID.String()),
		AccountNumber: accountNumber,
		Reason:        reason,
		ClosedAt:      closedAt,
	}
}
