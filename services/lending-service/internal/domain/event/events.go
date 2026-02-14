package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DomainEvent is the common interface every domain event satisfies.
type DomainEvent interface {
	EventID() string
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
	TenantID() string
}

// ---------------------------------------------------------------------------
// Base event (embedded by concrete events)
// ---------------------------------------------------------------------------

type baseEvent struct {
	ID          string    `json:"event_id"`
	Type        string    `json:"event_type"`
	Occurred    time.Time `json:"occurred_at"`
	AggregateId string    `json:"aggregate_id"`
	Tenant      string    `json:"tenant_id"`
}

func (e baseEvent) EventID() string      { return e.ID }
func (e baseEvent) EventType() string     { return e.Type }
func (e baseEvent) OccurredAt() time.Time { return e.Occurred }
func (e baseEvent) AggregateID() string   { return e.AggregateId }
func (e baseEvent) TenantID() string      { return e.Tenant }

func newBase(eventType, aggregateID, tenantID string, now time.Time) baseEvent {
	return baseEvent{
		ID:          uuid.New().String(),
		Type:        eventType,
		Occurred:    now,
		AggregateId: aggregateID,
		Tenant:      tenantID,
	}
}

// ---------------------------------------------------------------------------
// Loan Application Events
// ---------------------------------------------------------------------------

// LoanApplicationSubmitted is raised when a new application enters the system.
type LoanApplicationSubmitted struct {
	baseEvent
	ApplicantID     string          `json:"applicant_id"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	Currency        string          `json:"currency"`
	TermMonths      int             `json:"term_months"`
	Purpose         string          `json:"purpose"`
}

func NewLoanApplicationSubmitted(
	applicationID, tenantID, applicantID string,
	amount decimal.Decimal, currency string,
	termMonths int, purpose string, now time.Time,
) LoanApplicationSubmitted {
	return LoanApplicationSubmitted{
		baseEvent:       newBase("lending.loan_application.submitted", applicationID, tenantID, now),
		ApplicantID:     applicantID,
		RequestedAmount: amount,
		Currency:        currency,
		TermMonths:      termMonths,
		Purpose:         purpose,
	}
}

// LoanApplicationApproved is raised when an application is approved.
type LoanApplicationApproved struct {
	baseEvent
	ApplicantID string `json:"applicant_id"`
	Reason      string `json:"reason"`
	CreditScore string `json:"credit_score"`
}

func NewLoanApplicationApproved(
	applicationID, tenantID, applicantID, reason, creditScore string, now time.Time,
) LoanApplicationApproved {
	return LoanApplicationApproved{
		baseEvent:   newBase("lending.loan_application.approved", applicationID, tenantID, now),
		ApplicantID: applicantID,
		Reason:      reason,
		CreditScore: creditScore,
	}
}

// LoanApplicationRejected is raised when an application is rejected.
type LoanApplicationRejected struct {
	baseEvent
	ApplicantID string `json:"applicant_id"`
	Reason      string `json:"reason"`
}

func NewLoanApplicationRejected(
	applicationID, tenantID, applicantID, reason string, now time.Time,
) LoanApplicationRejected {
	return LoanApplicationRejected{
		baseEvent:   newBase("lending.loan_application.rejected", applicationID, tenantID, now),
		ApplicantID: applicantID,
		Reason:      reason,
	}
}

// ---------------------------------------------------------------------------
// Loan Events
// ---------------------------------------------------------------------------

// LoanDisbursed is raised when funds are disbursed to the borrower.
type LoanDisbursed struct {
	baseEvent
	ApplicationID    string          `json:"application_id"`
	BorrowerAccount  string          `json:"borrower_account_id"`
	Principal        decimal.Decimal `json:"principal"`
	Currency         string          `json:"currency"`
	InterestRateBps  int             `json:"interest_rate_bps"`
	TermMonths       int             `json:"term_months"`
	NextPaymentDue   time.Time       `json:"next_payment_due"`
}

func NewLoanDisbursed(
	loanID, tenantID, applicationID, borrowerAccount string,
	principal decimal.Decimal, currency string,
	rateBps, termMonths int, nextPaymentDue time.Time, now time.Time,
) LoanDisbursed {
	return LoanDisbursed{
		baseEvent:       newBase("lending.loan.disbursed", loanID, tenantID, now),
		ApplicationID:   applicationID,
		BorrowerAccount: borrowerAccount,
		Principal:       principal,
		Currency:        currency,
		InterestRateBps: rateBps,
		TermMonths:      termMonths,
		NextPaymentDue:  nextPaymentDue,
	}
}

// PaymentReceived is raised when a payment is applied to a loan.
type PaymentReceived struct {
	baseEvent
	Amount             decimal.Decimal `json:"amount"`
	Currency           string          `json:"currency"`
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewPaymentReceived(
	loanID, tenantID string,
	amount decimal.Decimal, currency string,
	outstandingBalance decimal.Decimal, now time.Time,
) PaymentReceived {
	return PaymentReceived{
		baseEvent:          newBase("lending.loan.payment_received", loanID, tenantID, now),
		Amount:             amount,
		Currency:           currency,
		OutstandingBalance: outstandingBalance,
	}
}

// LoanDelinquent is raised when a loan becomes delinquent.
type LoanDelinquent struct {
	baseEvent
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewLoanDelinquent(loanID, tenantID string, outstanding decimal.Decimal, now time.Time) LoanDelinquent {
	return LoanDelinquent{
		baseEvent:          newBase("lending.loan.delinquent", loanID, tenantID, now),
		OutstandingBalance: outstanding,
	}
}

// LoanDefault is raised when a loan enters default.
type LoanDefault struct {
	baseEvent
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewLoanDefault(loanID, tenantID string, outstanding decimal.Decimal, now time.Time) LoanDefault {
	return LoanDefault{
		baseEvent:          newBase("lending.loan.default", loanID, tenantID, now),
		OutstandingBalance: outstanding,
	}
}

// LoanPaidOff is raised when a loan is fully paid off.
type LoanPaidOff struct {
	baseEvent
}

func NewLoanPaidOff(loanID, tenantID string, now time.Time) LoanPaidOff {
	return LoanPaidOff{
		baseEvent: newBase("lending.loan.paid_off", loanID, tenantID, now),
	}
}
