package dto

import (
	"time"

	"github.com/google/uuid"
)

// OpenAccountRequest is the DTO for creating a new customer account.
type OpenAccountRequest struct {
	TenantID               uuid.UUID `json:"tenant_id"`
	AccountType            string    `json:"account_type"`
	Currency               string    `json:"currency"`
	HolderFirstName        string    `json:"holder_first_name"`
	HolderLastName         string    `json:"holder_last_name"`
	HolderEmail            string    `json:"holder_email"`
	IdentityVerificationID uuid.UUID `json:"identity_verification_id"`
}

// OpenAccountResponse is the DTO returned after creating a new customer account.
type OpenAccountResponse struct {
	AccountID         uuid.UUID `json:"account_id"`
	AccountNumber     string    `json:"account_number"`
	Status            string    `json:"status"`
	LedgerAccountCode string    `json:"ledger_account_code"`
	CreatedAt         time.Time `json:"created_at"`
}

// GetAccountRequest is the DTO for retrieving a customer account.
type GetAccountRequest struct {
	AccountID uuid.UUID `json:"account_id"`
}

// AccountResponse is the DTO representing a customer account in responses.
type AccountResponse struct {
	AccountID         uuid.UUID `json:"account_id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	AccountNumber     string    `json:"account_number"`
	AccountType       string    `json:"account_type"`
	Status            string    `json:"status"`
	Currency          string    `json:"currency"`
	LedgerAccountCode string    `json:"ledger_account_code"`
	HolderID          uuid.UUID `json:"holder_id"`
	HolderFirstName   string    `json:"holder_first_name"`
	HolderLastName    string    `json:"holder_last_name"`
	HolderEmail       string    `json:"holder_email"`
	Version           int       `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FreezeAccountRequest is the DTO for freezing a customer account.
type FreezeAccountRequest struct {
	AccountID uuid.UUID `json:"account_id"`
	Reason    string    `json:"reason"`
}

// CloseAccountRequest is the DTO for closing a customer account.
type CloseAccountRequest struct {
	AccountID uuid.UUID `json:"account_id"`
	Reason    string    `json:"reason"`
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
