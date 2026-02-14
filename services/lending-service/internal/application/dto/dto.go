package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// SubmitApplicationRequest carries the data needed to submit a new loan application.
type SubmitApplicationRequest struct {
	TenantID        string          `json:"tenant_id"`
	ApplicantID     string          `json:"applicant_id"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	Currency        string          `json:"currency"`
	TermMonths      int             `json:"term_months"`
	Purpose         string          `json:"purpose"`
}

// DisburseLoanRequest carries the data needed to disburse an approved loan.
type DisburseLoanRequest struct {
	TenantID          string `json:"tenant_id"`
	ApplicationID     string `json:"application_id"`
	BorrowerAccountID string `json:"borrower_account_id"`
	InterestRateBps   int    `json:"interest_rate_bps"`
}

// MakePaymentRequest carries the data for a loan payment.
type MakePaymentRequest struct {
	TenantID string          `json:"tenant_id"`
	LoanID   string          `json:"loan_id"`
	Amount   decimal.Decimal `json:"amount"`
}

// GetLoanRequest identifies a loan to retrieve.
type GetLoanRequest struct {
	TenantID string `json:"tenant_id"`
	LoanID   string `json:"loan_id"`
}

// GetApplicationRequest identifies a loan application to retrieve.
type GetApplicationRequest struct {
	TenantID      string `json:"tenant_id"`
	ApplicationID string `json:"application_id"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// LoanApplicationResponse is the external representation of a loan application.
type LoanApplicationResponse struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	ApplicantID     string          `json:"applicant_id"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	Currency        string          `json:"currency"`
	TermMonths      int             `json:"term_months"`
	Purpose         string          `json:"purpose"`
	Status          string          `json:"status"`
	DecisionReason  string          `json:"decision_reason,omitempty"`
	CreditScore     string          `json:"credit_score,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// AmortizationEntryResponse represents a single amortization schedule entry.
type AmortizationEntryResponse struct {
	Period           int             `json:"period"`
	DueDate          time.Time       `json:"due_date"`
	Principal        decimal.Decimal `json:"principal"`
	Interest         decimal.Decimal `json:"interest"`
	Total            decimal.Decimal `json:"total"`
	RemainingBalance decimal.Decimal `json:"remaining_balance"`
}

// LoanResponse is the external representation of a loan.
type LoanResponse struct {
	ID                 string                      `json:"id"`
	TenantID           string                      `json:"tenant_id"`
	ApplicationID      string                      `json:"application_id"`
	BorrowerAccountID  string                      `json:"borrower_account_id"`
	Principal          decimal.Decimal              `json:"principal"`
	Currency           string                      `json:"currency"`
	InterestRateBps    int                         `json:"interest_rate_bps"`
	TermMonths         int                         `json:"term_months"`
	Status             string                      `json:"status"`
	OutstandingBalance decimal.Decimal              `json:"outstanding_balance"`
	NextPaymentDue     time.Time                   `json:"next_payment_due"`
	Schedule           []AmortizationEntryResponse  `json:"schedule,omitempty"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

// PaymentResponse is the external representation of a payment result.
type PaymentResponse struct {
	LoanID             string          `json:"loan_id"`
	AmountPaid         decimal.Decimal `json:"amount_paid"`
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
	LoanStatus         string          `json:"loan_status"`
}
