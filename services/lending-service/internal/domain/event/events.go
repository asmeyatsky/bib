package event

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
)

// DomainEvent is an alias for the shared pkg/events.DomainEvent interface.
type DomainEvent = events.DomainEvent

// ---------------------------------------------------------------------------
// Loan Application Events
// ---------------------------------------------------------------------------

// LoanApplicationSubmitted is raised when a new application enters the system.
type LoanApplicationSubmitted struct {
	events.BaseEvent
	ApplicantID     string          `json:"applicant_id"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	Currency        string          `json:"currency"`
	Purpose         string          `json:"purpose"`
	TermMonths      int             `json:"term_months"`
}

func NewLoanApplicationSubmitted(
	applicationID, tenantID, applicantID string,
	amount decimal.Decimal, currency string,
	termMonths int, purpose string, _ time.Time,
) LoanApplicationSubmitted {
	return LoanApplicationSubmitted{
		BaseEvent:       events.NewBaseEvent("lending.loan_application.submitted", applicationID, "LoanApplication", tenantID),
		ApplicantID:     applicantID,
		RequestedAmount: amount,
		Currency:        currency,
		TermMonths:      termMonths,
		Purpose:         purpose,
	}
}

// LoanApplicationApproved is raised when an application is approved.
type LoanApplicationApproved struct {
	events.BaseEvent
	ApplicantID string `json:"applicant_id"`
	Reason      string `json:"reason"`
	CreditScore string `json:"credit_score"`
}

func NewLoanApplicationApproved(
	applicationID, tenantID, applicantID, reason, creditScore string, _ time.Time,
) LoanApplicationApproved {
	return LoanApplicationApproved{
		BaseEvent:   events.NewBaseEvent("lending.loan_application.approved", applicationID, "LoanApplication", tenantID),
		ApplicantID: applicantID,
		Reason:      reason,
		CreditScore: creditScore,
	}
}

// LoanApplicationRejected is raised when an application is rejected.
type LoanApplicationRejected struct {
	events.BaseEvent
	ApplicantID string `json:"applicant_id"`
	Reason      string `json:"reason"`
}

func NewLoanApplicationRejected(
	applicationID, tenantID, applicantID, reason string, _ time.Time,
) LoanApplicationRejected {
	return LoanApplicationRejected{
		BaseEvent:   events.NewBaseEvent("lending.loan_application.rejected", applicationID, "LoanApplication", tenantID),
		ApplicantID: applicantID,
		Reason:      reason,
	}
}

// ---------------------------------------------------------------------------
// Loan Events
// ---------------------------------------------------------------------------

// LoanDisbursed is raised when funds are disbursed to the borrower.
type LoanDisbursed struct {
	NextPaymentDue time.Time `json:"next_payment_due"`
	events.BaseEvent
	ApplicationID   string          `json:"application_id"`
	BorrowerAccount string          `json:"borrower_account_id"`
	Principal       decimal.Decimal `json:"principal"`
	Currency        string          `json:"currency"`
	InterestRateBps int             `json:"interest_rate_bps"`
	TermMonths      int             `json:"term_months"`
}

func NewLoanDisbursed(
	loanID, tenantID, applicationID, borrowerAccount string,
	principal decimal.Decimal, currency string,
	rateBps, termMonths int, nextPaymentDue time.Time, _ time.Time,
) LoanDisbursed {
	return LoanDisbursed{
		BaseEvent:       events.NewBaseEvent("lending.loan.disbursed", loanID, "Loan", tenantID),
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
	events.BaseEvent
	Amount             decimal.Decimal `json:"amount"`
	Currency           string          `json:"currency"`
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewPaymentReceived(
	loanID, tenantID string,
	amount decimal.Decimal, currency string,
	outstandingBalance decimal.Decimal, _ time.Time,
) PaymentReceived {
	return PaymentReceived{
		BaseEvent:          events.NewBaseEvent("lending.loan.payment_received", loanID, "Loan", tenantID),
		Amount:             amount,
		Currency:           currency,
		OutstandingBalance: outstandingBalance,
	}
}

// LoanDelinquent is raised when a loan becomes delinquent.
type LoanDelinquent struct {
	events.BaseEvent
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewLoanDelinquent(loanID, tenantID string, outstanding decimal.Decimal, _ time.Time) LoanDelinquent {
	return LoanDelinquent{
		BaseEvent:          events.NewBaseEvent("lending.loan.delinquent", loanID, "Loan", tenantID),
		OutstandingBalance: outstanding,
	}
}

// LoanDefault is raised when a loan enters default.
type LoanDefault struct {
	events.BaseEvent
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
}

func NewLoanDefault(loanID, tenantID string, outstanding decimal.Decimal, _ time.Time) LoanDefault {
	return LoanDefault{
		BaseEvent:          events.NewBaseEvent("lending.loan.default", loanID, "Loan", tenantID),
		OutstandingBalance: outstanding,
	}
}

// LoanPaidOff is raised when a loan is fully paid off.
type LoanPaidOff struct {
	events.BaseEvent
}

func NewLoanPaidOff(loanID, tenantID string, _ time.Time) LoanPaidOff {
	return LoanPaidOff{
		BaseEvent: events.NewBaseEvent("lending.loan.paid_off", loanID, "Loan", tenantID),
	}
}
