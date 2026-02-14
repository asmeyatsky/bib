package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// --- Exchange Rate DTOs ---

// GetExchangeRateRequest is the input DTO for fetching an exchange rate.
type GetExchangeRateRequest struct {
	TenantID      uuid.UUID
	BaseCurrency  string
	QuoteCurrency string
}

// ExchangeRateResponse is the output DTO for exchange rate queries.
type ExchangeRateResponse struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	BaseCurrency  string
	QuoteCurrency string
	Rate          decimal.Decimal
	InverseRate   decimal.Decimal
	Provider      string
	EffectiveAt   time.Time
	ExpiresAt     time.Time
	Version       int
	CreatedAt     time.Time
}

// --- Convert Amount DTOs ---

// ConvertAmountRequest is the input DTO for currency conversion.
type ConvertAmountRequest struct {
	TenantID     uuid.UUID
	FromCurrency string
	ToCurrency   string
	Amount       decimal.Decimal
}

// ConvertAmountResponse is the output DTO for currency conversion.
type ConvertAmountResponse struct {
	FromCurrency    string
	ToCurrency      string
	OriginalAmount  decimal.Decimal
	ConvertedAmount decimal.Decimal
	Rate            decimal.Decimal
	InverseRate     decimal.Decimal
	Provider        string
	EffectiveAt     time.Time
}

// --- List Rates DTOs ---

// ListRatesRequest is the input DTO for listing exchange rates.
type ListRatesRequest struct {
	TenantID     uuid.UUID
	BaseCurrency string
	AsOf         time.Time
}

// ListRatesResponse is the output DTO for listing exchange rates.
type ListRatesResponse struct {
	Rates []ExchangeRateResponse
}

// --- Revaluation DTOs ---

// RevaluateRequest is the input DTO for running an FX revaluation.
type RevaluateRequest struct {
	TenantID           uuid.UUID
	FunctionalCurrency string
	Positions          []ForeignCurrencyPositionDTO
}

// ForeignCurrencyPositionDTO transfers a foreign-currency position.
type ForeignCurrencyPositionDTO struct {
	AccountCode string
	Currency    string
	Amount      decimal.Decimal
}

// RevaluateResponse is the output DTO for the revaluation result.
type RevaluateResponse struct {
	TenantID           uuid.UUID
	FunctionalCurrency string
	TotalGainLoss      decimal.Decimal
	Entries            []RevaluationEntryDTO
}

// RevaluationEntryDTO transfers a single revaluation line item.
type RevaluationEntryDTO struct {
	AccountCode        string
	OriginalCurrency   string
	FunctionalCurrency string
	OriginalAmount     decimal.Decimal
	RevaluedAmount     decimal.Decimal
	GainLoss           decimal.Decimal
	Rate               decimal.Decimal
}
