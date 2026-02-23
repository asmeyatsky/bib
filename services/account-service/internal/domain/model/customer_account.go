package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/account-service/internal/domain/event"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// AccountStatus represents the lifecycle state of a customer account.
type AccountStatus string

const (
	AccountStatusPending AccountStatus = "PENDING"
	AccountStatusActive  AccountStatus = "ACTIVE"
	AccountStatusFrozen  AccountStatus = "FROZEN"
	AccountStatusClosed  AccountStatus = "CLOSED"
)

// CustomerAccount is the main aggregate root for the account domain.
// It is immutable; all state transitions return a new instance.
type CustomerAccount struct {
	id                uuid.UUID
	tenantID          uuid.UUID
	accountNumber     valueobject.AccountNumber
	accountType       valueobject.AccountType
	status            AccountStatus
	currency          string
	holder            AccountHolder
	ledgerAccountCode string
	version           int
	createdAt         time.Time
	updatedAt         time.Time
	domainEvents      []events.DomainEvent
}

// NewCustomerAccount creates a new CustomerAccount in PENDING status.
// It emits an AccountOpened domain event.
func NewCustomerAccount(
	tenantID uuid.UUID,
	accountType valueobject.AccountType,
	currency string,
	holder AccountHolder,
) (CustomerAccount, error) {
	if tenantID == uuid.Nil {
		return CustomerAccount{}, fmt.Errorf("tenant ID is required")
	}
	if accountType.IsZero() {
		return CustomerAccount{}, fmt.Errorf("account type is required")
	}
	if currency == "" {
		return CustomerAccount{}, fmt.Errorf("currency is required")
	}
	if len(currency) != 3 {
		return CustomerAccount{}, fmt.Errorf("currency must be a 3-letter ISO code, got %q", currency)
	}

	now := time.Now()
	id := uuid.New()
	accountNumber := valueobject.NewAccountNumber()

	account := CustomerAccount{
		id:            id,
		tenantID:      tenantID,
		accountNumber: accountNumber,
		accountType:   accountType,
		status:        AccountStatusPending,
		currency:      currency,
		holder:        holder,
		version:       1,
		createdAt:     now,
		updatedAt:     now,
	}

	account.domainEvents = append(account.domainEvents, event.NewAccountOpened(
		id,
		tenantID,
		accountNumber.String(),
		accountType.String(),
		currency,
		holder.FullName(),
		holder.Email(),
	))

	return account, nil
}

// ReconstructCustomerAccount recreates a CustomerAccount from persisted data
// without validation or emitting events. Used by repository implementations.
func ReconstructCustomerAccount(
	id uuid.UUID,
	tenantID uuid.UUID,
	accountNumber valueobject.AccountNumber,
	accountType valueobject.AccountType,
	status AccountStatus,
	currency string,
	holder AccountHolder,
	ledgerAccountCode string,
	version int,
	createdAt time.Time,
	updatedAt time.Time,
) CustomerAccount {
	return CustomerAccount{
		id:                id,
		tenantID:          tenantID,
		accountNumber:     accountNumber,
		accountType:       accountType,
		status:            status,
		currency:          currency,
		holder:            holder,
		ledgerAccountCode: ledgerAccountCode,
		version:           version,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

// Activate transitions the account from PENDING to ACTIVE.
// Returns a new CustomerAccount with the updated status and an AccountActivated event.
func (a CustomerAccount) Activate(now time.Time) (CustomerAccount, error) {
	if a.status != AccountStatusPending {
		return CustomerAccount{}, fmt.Errorf("cannot activate account in %s status: must be PENDING", a.status)
	}

	updated := a.clone()
	updated.status = AccountStatusActive
	updated.updatedAt = now
	updated.version = a.version + 1

	updated.domainEvents = append(updated.domainEvents, event.NewAccountActivated(
		a.id,
		a.tenantID,
		a.accountNumber.String(),
		now,
	))

	return updated, nil
}

// Freeze transitions the account from ACTIVE to FROZEN.
// Returns a new CustomerAccount with the updated status and an AccountFrozen event.
func (a CustomerAccount) Freeze(reason string, now time.Time) (CustomerAccount, error) {
	if a.status != AccountStatusActive {
		return CustomerAccount{}, fmt.Errorf("cannot freeze account in %s status: must be ACTIVE", a.status)
	}
	if reason == "" {
		return CustomerAccount{}, fmt.Errorf("reason is required to freeze an account")
	}

	updated := a.clone()
	updated.status = AccountStatusFrozen
	updated.updatedAt = now
	updated.version = a.version + 1

	updated.domainEvents = append(updated.domainEvents, event.NewAccountFrozen(
		a.id,
		a.tenantID,
		a.accountNumber.String(),
		reason,
		now,
	))

	return updated, nil
}

// Unfreeze transitions the account from FROZEN to ACTIVE.
// Returns a new CustomerAccount with the updated status and an AccountUnfrozen event.
func (a CustomerAccount) Unfreeze(now time.Time) (CustomerAccount, error) {
	if a.status != AccountStatusFrozen {
		return CustomerAccount{}, fmt.Errorf("cannot unfreeze account in %s status: must be FROZEN", a.status)
	}

	updated := a.clone()
	updated.status = AccountStatusActive
	updated.updatedAt = now
	updated.version = a.version + 1

	updated.domainEvents = append(updated.domainEvents, event.NewAccountUnfrozen(
		a.id,
		a.tenantID,
		a.accountNumber.String(),
		now,
	))

	return updated, nil
}

// Close transitions the account from ACTIVE or FROZEN to CLOSED.
// Returns a new CustomerAccount with the updated status and an AccountClosed event.
func (a CustomerAccount) Close(reason string, now time.Time) (CustomerAccount, error) {
	if a.status != AccountStatusActive && a.status != AccountStatusFrozen {
		return CustomerAccount{}, fmt.Errorf("cannot close account in %s status: must be ACTIVE or FROZEN", a.status)
	}
	if reason == "" {
		return CustomerAccount{}, fmt.Errorf("reason is required to close an account")
	}

	updated := a.clone()
	updated.status = AccountStatusClosed
	updated.updatedAt = now
	updated.version = a.version + 1

	updated.domainEvents = append(updated.domainEvents, event.NewAccountClosed(
		a.id,
		a.tenantID,
		a.accountNumber.String(),
		reason,
		now,
	))

	return updated, nil
}

// AssignLedgerCode assigns a ledger account code to this account.
// Returns a new CustomerAccount with the ledger code set.
func (a CustomerAccount) AssignLedgerCode(code string, now time.Time) (CustomerAccount, error) {
	if code == "" {
		return CustomerAccount{}, fmt.Errorf("ledger account code is required")
	}
	if a.ledgerAccountCode != "" {
		return CustomerAccount{}, fmt.Errorf("ledger account code already assigned: %s", a.ledgerAccountCode)
	}

	updated := a.clone()
	updated.ledgerAccountCode = code
	updated.updatedAt = now
	return updated, nil
}

// --- Accessors ---

// ID returns the account's unique identifier.
func (a CustomerAccount) ID() uuid.UUID { return a.id }

// TenantID returns the tenant identifier.
func (a CustomerAccount) TenantID() uuid.UUID { return a.tenantID }

// AccountNumber returns the account number value object.
func (a CustomerAccount) AccountNumber() valueobject.AccountNumber { return a.accountNumber }

// AccountType returns the account type value object.
func (a CustomerAccount) AccountType() valueobject.AccountType { return a.accountType }

// Status returns the current account status.
func (a CustomerAccount) Status() AccountStatus { return a.status }

// Currency returns the account currency code.
func (a CustomerAccount) Currency() string { return a.currency }

// Holder returns the account holder entity.
func (a CustomerAccount) Holder() AccountHolder { return a.holder }

// LedgerAccountCode returns the linked ledger account code.
func (a CustomerAccount) LedgerAccountCode() string { return a.ledgerAccountCode }

// Version returns the current version for optimistic concurrency.
func (a CustomerAccount) Version() int { return a.version }

// CreatedAt returns the account creation timestamp.
func (a CustomerAccount) CreatedAt() time.Time { return a.createdAt }

// UpdatedAt returns the last update timestamp.
func (a CustomerAccount) UpdatedAt() time.Time { return a.updatedAt }

// DomainEvents returns all uncommitted domain events.
func (a CustomerAccount) DomainEvents() []events.DomainEvent {
	events := make([]events.DomainEvent, len(a.domainEvents))
	copy(events, a.domainEvents)
	return events
}

// ClearDomainEvents returns a new CustomerAccount with domain events cleared.
func (a CustomerAccount) ClearDomainEvents() CustomerAccount {
	updated := a.clone()
	updated.domainEvents = nil
	return updated
}

// clone creates a shallow copy of the account for immutability.
func (a CustomerAccount) clone() CustomerAccount {
	cloned := a
	if len(a.domainEvents) > 0 {
		cloned.domainEvents = make([]events.DomainEvent, len(a.domainEvents))
		copy(cloned.domainEvents, a.domainEvents)
	}
	return cloned
}
