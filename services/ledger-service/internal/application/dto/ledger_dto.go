package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PostJournalEntryRequest is the input DTO for posting a journal entry.
type PostJournalEntryRequest struct {
	EffectiveDate time.Time
	Description   string
	Reference     string
	Postings      []PostingPairDTO
	TenantID      uuid.UUID
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
	EffectiveDate time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Status        string
	Description   string
	Reference     string
	Postings      []PostingPairDTO
	Version       int
	ID            uuid.UUID
	TenantID      uuid.UUID
}

// GetBalanceRequest is the input DTO for balance queries.
type GetBalanceRequest struct {
	AsOf        time.Time
	AccountCode string
	Currency    string
}

// BalanceResponse is the output DTO for balance queries.
type BalanceResponse struct {
	AsOf        time.Time
	AccountCode string
	Amount      decimal.Decimal
	Currency    string
}

// BackvalueEntryRequest is the input DTO for back-valuation.
type BackvalueEntryRequest struct {
	NewDate time.Time
	EntryID uuid.UUID
}

// PeriodCloseRequest is the input DTO for closing a fiscal period.
type PeriodCloseRequest struct {
	TenantID uuid.UUID
	Year     int
	Month    int
}

// ListEntriesRequest is the input DTO for listing journal entries.
type ListEntriesRequest struct {
	FromDate    time.Time
	ToDate      time.Time
	AccountCode string
	PageSize    int
	Offset      int
	TenantID    uuid.UUID
}

// ListEntriesResponse is the output DTO for listing journal entries.
type ListEntriesResponse struct {
	Entries    []JournalEntryResponse
	TotalCount int
}
