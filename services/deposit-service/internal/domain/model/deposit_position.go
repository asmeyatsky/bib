package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/event"
)

// PositionStatus represents the lifecycle state of a deposit position.
type PositionStatus string

const (
	PositionStatusActive  PositionStatus = "ACTIVE"
	PositionStatusMatured PositionStatus = "MATURED"
	PositionStatusClosed  PositionStatus = "CLOSED"
)

// DepositPosition is the aggregate root for a customer's deposit holding.
// It tracks principal, accrued interest, status, and lifecycle transitions.
type DepositPosition struct {
	openedAt        time.Time
	updatedAt       time.Time
	createdAt       time.Time
	lastAccrualDate time.Time
	maturityDate    *time.Time
	accruedInterest decimal.Decimal
	status          PositionStatus
	currency        string
	principal       decimal.Decimal
	domainEvents    []events.DomainEvent
	version         int
	id              uuid.UUID
	productID       uuid.UUID
	accountID       uuid.UUID
	tenantID        uuid.UUID
}

// NewDepositPosition creates a new deposit position in ACTIVE status.
func NewDepositPosition(
	tenantID, accountID, productID uuid.UUID,
	principal decimal.Decimal,
	currency string,
	maturityDate *time.Time,
) (DepositPosition, error) {
	if tenantID == uuid.Nil {
		return DepositPosition{}, fmt.Errorf("tenant ID is required")
	}
	if accountID == uuid.Nil {
		return DepositPosition{}, fmt.Errorf("account ID is required")
	}
	if productID == uuid.Nil {
		return DepositPosition{}, fmt.Errorf("product ID is required")
	}
	if principal.LessThanOrEqual(decimal.Zero) {
		return DepositPosition{}, fmt.Errorf("principal must be positive")
	}
	if currency == "" || len(currency) != 3 {
		return DepositPosition{}, fmt.Errorf("currency must be a 3-letter ISO code")
	}

	now := time.Now().UTC()
	positionID := uuid.New()

	pos := DepositPosition{
		id:              positionID,
		tenantID:        tenantID,
		accountID:       accountID,
		productID:       productID,
		principal:       principal,
		currency:        currency,
		accruedInterest: decimal.Zero,
		status:          PositionStatusActive,
		openedAt:        now,
		maturityDate:    maturityDate,
		lastAccrualDate: now,
		version:         1,
		createdAt:       now,
		updatedAt:       now,
	}

	pos.domainEvents = append(pos.domainEvents,
		event.NewDepositOpened(positionID, tenantID, accountID, productID, principal, currency),
	)

	return pos, nil
}

// ReconstructPosition recreates a DepositPosition from persistence (no validation, no events).
func ReconstructPosition(
	id, tenantID, accountID, productID uuid.UUID,
	principal decimal.Decimal,
	currency string,
	accruedInterest decimal.Decimal,
	status PositionStatus,
	openedAt time.Time,
	maturityDate *time.Time,
	lastAccrualDate time.Time,
	version int,
	createdAt, updatedAt time.Time,
) DepositPosition {
	return DepositPosition{
		id:              id,
		tenantID:        tenantID,
		accountID:       accountID,
		productID:       productID,
		principal:       principal,
		currency:        currency,
		accruedInterest: accruedInterest,
		status:          status,
		openedAt:        openedAt,
		maturityDate:    maturityDate,
		lastAccrualDate: lastAccrualDate,
		version:         version,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// AccrueInterest calculates and adds interest for the days elapsed since the last accrual.
// The interest formula is: principal * dailyRate * days. This is immutable - returns a new copy.
func (p DepositPosition) AccrueInterest(dailyRate decimal.Decimal, asOf time.Time) (DepositPosition, error) {
	if p.status != PositionStatusActive {
		return DepositPosition{}, fmt.Errorf("can only accrue interest on ACTIVE positions, current: %s", p.status)
	}
	if asOf.Before(p.lastAccrualDate) {
		return DepositPosition{}, fmt.Errorf("accrual date %s is before last accrual date %s", asOf, p.lastAccrualDate)
	}

	// Calculate number of days since last accrual
	days := daysBetween(p.lastAccrualDate, asOf)
	if days == 0 {
		return p, nil // no days to accrue
	}

	// Interest = principal * dailyRate * days
	daysDecimal := decimal.NewFromInt(int64(days))
	interest := p.principal.Mul(dailyRate).Mul(daysDecimal)

	// Round to 4 decimal places (standard for monetary calculations)
	interest = interest.Round(4)

	accrued := p
	accrued.accruedInterest = p.accruedInterest.Add(interest)
	accrued.lastAccrualDate = asOf
	accrued.updatedAt = asOf
	accrued.version++
	accrued.domainEvents = append(copyEvents(p.domainEvents),
		event.NewInterestAccrued(p.id, p.tenantID, p.accountID, interest, p.currency, asOf),
	)

	return accrued, nil
}

// Mature transitions the position from ACTIVE to MATURED (immutable - returns new copy).
func (p DepositPosition) Mature(now time.Time) (DepositPosition, error) {
	if p.status != PositionStatusActive {
		return DepositPosition{}, fmt.Errorf("can only mature ACTIVE positions, current: %s", p.status)
	}

	matured := p
	matured.status = PositionStatusMatured
	matured.updatedAt = now
	matured.version++
	matured.domainEvents = append(copyEvents(p.domainEvents),
		event.NewDepositMatured(p.id, p.tenantID, p.accountID),
	)

	return matured, nil
}

// Close transitions the position from ACTIVE or MATURED to CLOSED (immutable - returns new copy).
func (p DepositPosition) Close(now time.Time) (DepositPosition, error) {
	if p.status != PositionStatusActive && p.status != PositionStatusMatured {
		return DepositPosition{}, fmt.Errorf("can only close ACTIVE or MATURED positions, current: %s", p.status)
	}

	closed := p
	closed.status = PositionStatusClosed
	closed.updatedAt = now
	closed.version++
	closed.domainEvents = append(copyEvents(p.domainEvents),
		event.NewDepositClosed(p.id, p.tenantID, p.accountID),
	)

	return closed, nil
}

// TotalBalance returns principal + accrued interest.
func (p DepositPosition) TotalBalance() decimal.Decimal {
	return p.principal.Add(p.accruedInterest)
}

// Accessors
func (p DepositPosition) ID() uuid.UUID                      { return p.id }
func (p DepositPosition) TenantID() uuid.UUID                { return p.tenantID }
func (p DepositPosition) AccountID() uuid.UUID               { return p.accountID }
func (p DepositPosition) ProductID() uuid.UUID               { return p.productID }
func (p DepositPosition) Principal() decimal.Decimal         { return p.principal }
func (p DepositPosition) Currency() string                   { return p.currency }
func (p DepositPosition) AccruedInterest() decimal.Decimal   { return p.accruedInterest }
func (p DepositPosition) Status() PositionStatus             { return p.status }
func (p DepositPosition) OpenedAt() time.Time                { return p.openedAt }
func (p DepositPosition) MaturityDate() *time.Time           { return p.maturityDate }
func (p DepositPosition) LastAccrualDate() time.Time         { return p.lastAccrualDate }
func (p DepositPosition) Version() int                       { return p.version }
func (p DepositPosition) CreatedAt() time.Time               { return p.createdAt }
func (p DepositPosition) UpdatedAt() time.Time               { return p.updatedAt }
func (p DepositPosition) DomainEvents() []events.DomainEvent { return p.domainEvents }

// ClearDomainEvents returns the collected domain events.
func (p DepositPosition) ClearDomainEvents() []events.DomainEvent {
	return p.domainEvents
}

// daysBetween calculates the number of calendar days between two times.
func daysBetween(from, to time.Time) int {
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	duration := toDate.Sub(fromDate)
	return int(duration.Hours() / 24)
}

// copyEvents creates a defensive copy of an event slice.
func copyEvents(evts []events.DomainEvent) []events.DomainEvent {
	if evts == nil {
		return nil
	}
	c := make([]events.DomainEvent, len(evts))
	copy(c, evts)
	return c
}
