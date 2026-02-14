package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PostJournalEntryRequest is the input DTO for posting a journal entry.
type PostJournalEntryRequest struct {
	TenantID      uuid.UUID
	EffectiveDate time.Time
	Postings      []PostingPairDTO
	Description   string
	Reference     string
}

// PostingPairDTO transfers posting pair data.
type PostingPairDTO struct {
	DebitAccount  string
	CreditAccount string
	Amount        decimal.Decimal
	Currency      string
	Description   string
}

// JournalEntryResponse is the output DTO for a journal entry.
type JournalEntryResponse struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	EffectiveDate time.Time
	Postings      []PostingPairDTO
	Status        string
	Description   string
	Reference     string
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GetBalanceRequest is the input DTO for balance queries.
type GetBalanceRequest struct {
	AccountCode string
	Currency    string
	AsOf        time.Time
}

// BalanceResponse is the output DTO for balance queries.
type BalanceResponse struct {
	AccountCode string
	Amount      decimal.Decimal
	Currency    string
	AsOf        time.Time
}

// BackvalueEntryRequest is the input DTO for back-valuation.
type BackvalueEntryRequest struct {
	EntryID uuid.UUID
	NewDate time.Time
}

// PeriodCloseRequest is the input DTO for closing a fiscal period.
type PeriodCloseRequest struct {
	TenantID uuid.UUID
	Year     int
	Month    int
}

// ListEntriesRequest is the input DTO for listing journal entries.
type ListEntriesRequest struct {
	TenantID    uuid.UUID
	AccountCode string
	FromDate    time.Time
	ToDate      time.Time
	PageSize    int
	Offset      int
}

// ListEntriesResponse is the output DTO for listing journal entries.
type ListEntriesResponse struct {
	Entries    []JournalEntryResponse
	TotalCount int
}
