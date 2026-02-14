package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

// ---------------------------------------------------------------------------
// Loan aggregate root (Loan Servicing System)
// ---------------------------------------------------------------------------

// Loan is an immutable aggregate. Mutations return a new copy.
type Loan struct {
	id                string
	tenantID          string
	applicationID     string
	borrowerAccountID string
	principal         decimal.Decimal
	currency          string
	interestRateBps   int
	termMonths        int
	status            valueobject.LoanStatus
	schedule          []AmortizationEntry
	outstandingBalance decimal.Decimal
	nextPaymentDue    time.Time
	version           int
	createdAt         time.Time
	updatedAt         time.Time
	domainEvents      []event.DomainEvent
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewLoan creates a loan from an approved application and generates the
// amortization schedule. The loan starts in ACTIVE status.
func NewLoan(
	tenantID, applicationID, borrowerAccountID string,
	principal decimal.Decimal,
	currency string,
	interestRateBps, termMonths int,
	now time.Time,
) (Loan, error) {
	if tenantID == "" {
		return Loan{}, errors.New("tenant ID is required")
	}
	if applicationID == "" {
		return Loan{}, errors.New("application ID is required")
	}
	if borrowerAccountID == "" {
		return Loan{}, errors.New("borrower account ID is required")
	}
	if principal.LessThanOrEqual(decimal.Zero) {
		return Loan{}, errors.New("principal must be positive")
	}
	if currency == "" {
		return Loan{}, errors.New("currency is required")
	}
	if termMonths <= 0 {
		return Loan{}, errors.New("term months must be positive")
	}

	id := uuid.New().String()
	sched := GenerateAmortizationSchedule(principal, interestRateBps, termMonths, now)

	var nextDue time.Time
	if len(sched) > 0 {
		nextDue = sched[0].DueDate
	}

	loan := Loan{
		id:                 id,
		tenantID:           tenantID,
		applicationID:      applicationID,
		borrowerAccountID:  borrowerAccountID,
		principal:          principal,
		currency:           currency,
		interestRateBps:    interestRateBps,
		termMonths:         termMonths,
		status:             valueobject.LoanStatusActive,
		schedule:           sched,
		outstandingBalance: principal,
		nextPaymentDue:     nextDue,
		version:            1,
		createdAt:          now,
		updatedAt:          now,
	}

	loan.domainEvents = append(loan.domainEvents, event.NewLoanDisbursed(
		id, tenantID, applicationID, borrowerAccountID,
		principal, currency, interestRateBps, termMonths, nextDue, now,
	))

	return loan, nil
}

// ReconstructLoan rebuilds a Loan aggregate from persistence.
func ReconstructLoan(
	id, tenantID, applicationID, borrowerAccountID string,
	principal decimal.Decimal,
	currency string,
	interestRateBps, termMonths int,
	status valueobject.LoanStatus,
	schedule []AmortizationEntry,
	outstandingBalance decimal.Decimal,
	nextPaymentDue time.Time,
	version int,
	createdAt, updatedAt time.Time,
) Loan {
	return Loan{
		id:                 id,
		tenantID:           tenantID,
		applicationID:      applicationID,
		borrowerAccountID:  borrowerAccountID,
		principal:          principal,
		currency:           currency,
		interestRateBps:    interestRateBps,
		termMonths:         termMonths,
		status:             status,
		schedule:           schedule,
		outstandingBalance: outstandingBalance,
		nextPaymentDue:     nextPaymentDue,
		version:            version,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
	}
}

// ---------------------------------------------------------------------------
// State transitions
// ---------------------------------------------------------------------------

// MakePayment reduces the outstanding balance and emits PaymentReceived.
func (l Loan) MakePayment(amount decimal.Decimal, now time.Time) (Loan, error) {
	if !l.status.Equal(valueobject.LoanStatusActive) && !l.status.Equal(valueobject.LoanStatusDelinquent) {
		return l, errors.New("payments can only be made on active or delinquent loans")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return l, errors.New("payment amount must be positive")
	}
	if amount.GreaterThan(l.outstandingBalance) {
		return l, errors.New("payment exceeds outstanding balance")
	}

	next := l
	next.outstandingBalance = l.outstandingBalance.Sub(amount)
	next.updatedAt = now
	next.domainEvents = copyEvents(l.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewPaymentReceived(
		l.id, l.tenantID, amount, l.currency, next.outstandingBalance, now,
	))

	// If fully paid off, transition to PAID_OFF.
	if next.outstandingBalance.Equal(decimal.Zero) {
		next.status = valueobject.LoanStatusPaidOff
		next.domainEvents = append(next.domainEvents, event.NewLoanPaidOff(l.id, l.tenantID, now))
	}

	return next, nil
}

// MarkDelinquent transitions ACTIVE -> DELINQUENT.
func (l Loan) MarkDelinquent(now time.Time) (Loan, error) {
	if !l.status.Equal(valueobject.LoanStatusActive) {
		return l, valueobject.ErrInvalidStatusTransition
	}
	next := l
	next.status = valueobject.LoanStatusDelinquent
	next.updatedAt = now
	next.domainEvents = copyEvents(l.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewLoanDelinquent(l.id, l.tenantID, l.outstandingBalance, now))
	return next, nil
}

// MarkDefault transitions DELINQUENT -> DEFAULT.
func (l Loan) MarkDefault(now time.Time) (Loan, error) {
	if !l.status.Equal(valueobject.LoanStatusDelinquent) {
		return l, valueobject.ErrInvalidStatusTransition
	}
	next := l
	next.status = valueobject.LoanStatusDefault
	next.updatedAt = now
	next.domainEvents = copyEvents(l.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewLoanDefault(l.id, l.tenantID, l.outstandingBalance, now))
	return next, nil
}

// PayOff sets the outstanding balance to zero and transitions to PAID_OFF.
func (l Loan) PayOff(now time.Time) (Loan, error) {
	if l.status.Equal(valueobject.LoanStatusPaidOff) || l.status.Equal(valueobject.LoanStatusWrittenOff) {
		return l, valueobject.ErrInvalidStatusTransition
	}
	next := l
	next.outstandingBalance = decimal.Zero
	next.status = valueobject.LoanStatusPaidOff
	next.updatedAt = now
	next.domainEvents = copyEvents(l.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewLoanPaidOff(l.id, l.tenantID, now))
	return next, nil
}

// WriteOff transitions DEFAULT -> WRITTEN_OFF.
func (l Loan) WriteOff(now time.Time) (Loan, error) {
	if !l.status.Equal(valueobject.LoanStatusDefault) {
		return l, valueobject.ErrInvalidStatusTransition
	}
	next := l
	next.status = valueobject.LoanStatusWrittenOff
	next.updatedAt = now
	next.domainEvents = copyEvents(l.domainEvents)
	return next, nil
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (l Loan) ID() string                          { return l.id }
func (l Loan) TenantID() string                    { return l.tenantID }
func (l Loan) ApplicationID() string               { return l.applicationID }
func (l Loan) BorrowerAccountID() string           { return l.borrowerAccountID }
func (l Loan) Principal() decimal.Decimal           { return l.principal }
func (l Loan) Currency() string                    { return l.currency }
func (l Loan) InterestRateBps() int                { return l.interestRateBps }
func (l Loan) TermMonths() int                     { return l.termMonths }
func (l Loan) Status() valueobject.LoanStatus      { return l.status }
func (l Loan) OutstandingBalance() decimal.Decimal  { return l.outstandingBalance }
func (l Loan) NextPaymentDue() time.Time           { return l.nextPaymentDue }
func (l Loan) Version() int                        { return l.version }
func (l Loan) CreatedAt() time.Time                { return l.createdAt }
func (l Loan) UpdatedAt() time.Time                { return l.updatedAt }
func (l Loan) DomainEvents() []event.DomainEvent   { return l.domainEvents }

// Schedule returns a defensive copy of the amortization schedule.
func (l Loan) Schedule() []AmortizationEntry {
	if l.schedule == nil {
		return nil
	}
	out := make([]AmortizationEntry, len(l.schedule))
	copy(out, l.schedule)
	return out
}

// ClearEvents returns a copy with an empty event list.
func (l Loan) ClearEvents() Loan {
	next := l
	next.domainEvents = nil
	return next
}
