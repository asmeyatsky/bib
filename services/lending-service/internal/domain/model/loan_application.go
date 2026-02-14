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
// LoanApplication aggregate root (Loan Origination System)
// ---------------------------------------------------------------------------

// LoanApplication is an immutable aggregate. Every mutation returns a new copy.
type LoanApplication struct {
	id              string
	tenantID        string
	applicantID     string
	requestedAmount decimal.Decimal
	currency        string
	termMonths      int
	purpose         string
	status          valueobject.LoanApplicationStatus
	decisionReason  string
	creditScore     string
	version         int
	createdAt       time.Time
	updatedAt       time.Time
	domainEvents    []event.DomainEvent
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewLoanApplication creates a brand-new application in SUBMITTED status.
func NewLoanApplication(
	tenantID, applicantID string,
	requestedAmount decimal.Decimal,
	currency string,
	termMonths int,
	purpose string,
	now time.Time,
) (LoanApplication, error) {
	if tenantID == "" {
		return LoanApplication{}, errors.New("tenant ID is required")
	}
	if applicantID == "" {
		return LoanApplication{}, errors.New("applicant ID is required")
	}
	if requestedAmount.LessThanOrEqual(decimal.Zero) {
		return LoanApplication{}, errors.New("requested amount must be positive")
	}
	if currency == "" {
		return LoanApplication{}, errors.New("currency is required")
	}
	if termMonths <= 0 {
		return LoanApplication{}, errors.New("term months must be positive")
	}

	id := uuid.New().String()
	app := LoanApplication{
		id:              id,
		tenantID:        tenantID,
		applicantID:     applicantID,
		requestedAmount: requestedAmount,
		currency:        currency,
		termMonths:      termMonths,
		purpose:         purpose,
		status:          valueobject.LoanApplicationStatusSubmitted,
		version:         1,
		createdAt:       now,
		updatedAt:       now,
	}

	submitted := event.NewLoanApplicationSubmitted(
		id, tenantID, applicantID, requestedAmount, currency, termMonths, purpose, now,
	)
	app.domainEvents = append(app.domainEvents, submitted)
	return app, nil
}

// ReconstructLoanApplication rebuilds an aggregate from persistence without side-effects.
func ReconstructLoanApplication(
	id, tenantID, applicantID string,
	requestedAmount decimal.Decimal,
	currency string,
	termMonths int,
	purpose string,
	status valueobject.LoanApplicationStatus,
	decisionReason, creditScore string,
	version int,
	createdAt, updatedAt time.Time,
) LoanApplication {
	return LoanApplication{
		id:              id,
		tenantID:        tenantID,
		applicantID:     applicantID,
		requestedAmount: requestedAmount,
		currency:        currency,
		termMonths:      termMonths,
		purpose:         purpose,
		status:          status,
		decisionReason:  decisionReason,
		creditScore:     creditScore,
		version:         version,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// ---------------------------------------------------------------------------
// State transitions (each returns a new copy)
// ---------------------------------------------------------------------------

// SubmitForReview transitions SUBMITTED -> UNDER_REVIEW.
func (a LoanApplication) SubmitForReview(now time.Time) (LoanApplication, error) {
	if !a.status.Equal(valueobject.LoanApplicationStatusSubmitted) {
		return a, valueobject.ErrInvalidStatusTransition
	}
	next := a
	next.status = valueobject.LoanApplicationStatusUnderReview
	next.updatedAt = now
	next.domainEvents = copyEvents(a.domainEvents)
	return next, nil
}

// Approve transitions UNDER_REVIEW -> APPROVED and emits LoanApplicationApproved.
func (a LoanApplication) Approve(reason, creditScore string, now time.Time) (LoanApplication, error) {
	if !a.status.Equal(valueobject.LoanApplicationStatusUnderReview) {
		return a, valueobject.ErrInvalidStatusTransition
	}
	next := a
	next.status = valueobject.LoanApplicationStatusApproved
	next.decisionReason = reason
	next.creditScore = creditScore
	next.updatedAt = now
	next.domainEvents = copyEvents(a.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewLoanApplicationApproved(
		a.id, a.tenantID, a.applicantID, reason, creditScore, now,
	))
	return next, nil
}

// Reject transitions UNDER_REVIEW -> REJECTED and emits LoanApplicationRejected.
func (a LoanApplication) Reject(reason string, now time.Time) (LoanApplication, error) {
	if !a.status.Equal(valueobject.LoanApplicationStatusUnderReview) {
		return a, valueobject.ErrInvalidStatusTransition
	}
	next := a
	next.status = valueobject.LoanApplicationStatusRejected
	next.decisionReason = reason
	next.updatedAt = now
	next.domainEvents = copyEvents(a.domainEvents)
	next.domainEvents = append(next.domainEvents, event.NewLoanApplicationRejected(
		a.id, a.tenantID, a.applicantID, reason, now,
	))
	return next, nil
}

// MarkDisbursed transitions APPROVED -> DISBURSED.
func (a LoanApplication) MarkDisbursed(now time.Time) (LoanApplication, error) {
	if !a.status.Equal(valueobject.LoanApplicationStatusApproved) {
		return a, valueobject.ErrInvalidStatusTransition
	}
	next := a
	next.status = valueobject.LoanApplicationStatusDisbursed
	next.updatedAt = now
	next.domainEvents = copyEvents(a.domainEvents)
	return next, nil
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (a LoanApplication) ID() string                                   { return a.id }
func (a LoanApplication) TenantID() string                             { return a.tenantID }
func (a LoanApplication) ApplicantID() string                          { return a.applicantID }
func (a LoanApplication) RequestedAmount() decimal.Decimal              { return a.requestedAmount }
func (a LoanApplication) Currency() string                             { return a.currency }
func (a LoanApplication) TermMonths() int                              { return a.termMonths }
func (a LoanApplication) Purpose() string                              { return a.purpose }
func (a LoanApplication) Status() valueobject.LoanApplicationStatus    { return a.status }
func (a LoanApplication) DecisionReason() string                       { return a.decisionReason }
func (a LoanApplication) CreditScore() string                          { return a.creditScore }
func (a LoanApplication) Version() int                                 { return a.version }
func (a LoanApplication) CreatedAt() time.Time                         { return a.createdAt }
func (a LoanApplication) UpdatedAt() time.Time                         { return a.updatedAt }
func (a LoanApplication) DomainEvents() []event.DomainEvent            { return a.domainEvents }

// ClearEvents returns a copy with an empty event list (call after publishing).
func (a LoanApplication) ClearEvents() LoanApplication {
	next := a
	next.domainEvents = nil
	return next
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func copyEvents(src []event.DomainEvent) []event.DomainEvent {
	if len(src) == 0 {
		return nil
	}
	dst := make([]event.DomainEvent, len(src))
	copy(dst, src)
	return dst
}
