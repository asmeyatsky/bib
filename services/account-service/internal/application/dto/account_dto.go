package dto

import (
	"time"

	"github.com/google/uuid"
)

// OpenAccountRequest is the DTO for creating a new customer account.
type OpenAccountRequest struct {
	AccountType            string    `json:"account_type"`
	Currency               string    `json:"currency"`
	HolderFirstName        string    `json:"holder_first_name"`
	HolderLastName         string    `json:"holder_last_name"`
	HolderEmail            string    `json:"holder_email"`
	TenantID               uuid.UUID `json:"tenant_id"`
	IdentityVerificationID uuid.UUID `json:"identity_verification_id"`
}

// OpenAccountResponse is the DTO returned after creating a new customer account.
type OpenAccountResponse struct {
	CreatedAt         time.Time `json:"created_at"`
	AccountNumber     string    `json:"account_number"`
	Status            string    `json:"status"`
	LedgerAccountCode string    `json:"ledger_account_code"`
	AccountID         uuid.UUID `json:"account_id"`
}

// GetAccountRequest is the DTO for retrieving a customer account.
type GetAccountRequest struct {
	AccountID uuid.UUID `json:"account_id"`
}

// AccountResponse is the DTO representing a customer account in responses.
type AccountResponse struct {
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LedgerAccountCode string    `json:"ledger_account_code"`
	AccountType       string    `json:"account_type"`
	Status            string    `json:"status"`
	Currency          string    `json:"currency"`
	HolderFirstName   string    `json:"holder_first_name"`
	HolderLastName    string    `json:"holder_last_name"`
	HolderEmail       string    `json:"holder_email"`
	AccountNumber     string    `json:"account_number"`
	Version           int       `json:"version"`
	AccountID         uuid.UUID `json:"account_id"`
	HolderID          uuid.UUID `json:"holder_id"`
	TenantID          uuid.UUID `json:"tenant_id"`
}

// FreezeAccountRequest is the DTO for freezing a customer account.
type FreezeAccountRequest struct {
	Reason    string    `json:"reason"`
	AccountID uuid.UUID `json:"account_id"`
}

// CloseAccountRequest is the DTO for closing a customer account.
type CloseAccountRequest struct {
	Reason    string    `json:"reason"`
	AccountID uuid.UUID `json:"account_id"`
}

// ListAccountsRequest is the DTO for listing customer accounts with pagination.
type ListAccountsRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	HolderID uuid.UUID `json:"holder_id"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

// ListAccountsResponse is the DTO returned when listing customer accounts.
type ListAccountsResponse struct {
	Accounts   []AccountResponse `json:"accounts"`
	TotalCount int               `json:"total_count"`
}
