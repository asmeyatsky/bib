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
	Purpose         string          `json:"purpose"`
	TermMonths      int             `json:"term_months"`
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
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	ApplicantID     string          `json:"applicant_id"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	Currency        string          `json:"currency"`
	Purpose         string          `json:"purpose"`
	Status          string          `json:"status"`
	DecisionReason  string          `json:"decision_reason,omitempty"`
	CreditScore     string          `json:"credit_score,omitempty"`
	TermMonths      int             `json:"term_months"`
}

// AmortizationEntryResponse represents a single amortization schedule entry.
type AmortizationEntryResponse struct {
	DueDate          time.Time       `json:"due_date"`
	Principal        decimal.Decimal `json:"principal"`
	Interest         decimal.Decimal `json:"interest"`
	Total            decimal.Decimal `json:"total"`
	RemainingBalance decimal.Decimal `json:"remaining_balance"`
	Period           int             `json:"period"`
}

// LoanResponse is the external representation of a loan.
type LoanResponse struct {
	NextPaymentDue     time.Time                   `json:"next_payment_due"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	CreatedAt          time.Time                   `json:"created_at"`
	OutstandingBalance decimal.Decimal             `json:"outstanding_balance"`
	Principal          decimal.Decimal             `json:"principal"`
	Currency           string                      `json:"currency"`
	Status             string                      `json:"status"`
	ID                 string                      `json:"id"`
	BorrowerAccountID  string                      `json:"borrower_account_id"`
	ApplicationID      string                      `json:"application_id"`
	TenantID           string                      `json:"tenant_id"`
	Schedule           []AmortizationEntryResponse `json:"schedule,omitempty"`
	InterestRateBps    int                         `json:"interest_rate_bps"`
	TermMonths         int                         `json:"term_months"`
}

// PaymentResponse is the external representation of a payment result.
type PaymentResponse struct {
	LoanID             string          `json:"loan_id"`
	AmountPaid         decimal.Decimal `json:"amount_paid"`
	OutstandingBalance decimal.Decimal `json:"outstanding_balance"`
	LoanStatus         string          `json:"loan_status"`
}
