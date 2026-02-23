package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// --- Exchange Rate DTOs ---

// GetExchangeRateRequest is the input DTO for fetching an exchange rate.
type GetExchangeRateRequest struct {
	BaseCurrency  string
	QuoteCurrency string
	TenantID      uuid.UUID
}

// ExchangeRateResponse is the output DTO for exchange rate queries.
type ExchangeRateResponse struct {
	EffectiveAt   time.Time
	ExpiresAt     time.Time
	CreatedAt     time.Time
	BaseCurrency  string
	QuoteCurrency string
	Rate          decimal.Decimal
	InverseRate   decimal.Decimal
	Provider      string
	Version       int
	ID            uuid.UUID
	TenantID      uuid.UUID
}

// --- Convert Amount DTOs ---

// ConvertAmountRequest is the input DTO for currency conversion.
type ConvertAmountRequest struct {
	FromCurrency string
	ToCurrency   string
	Amount       decimal.Decimal
	TenantID     uuid.UUID
}

// ConvertAmountResponse is the output DTO for currency conversion.
type ConvertAmountResponse struct {
	EffectiveAt     time.Time
	FromCurrency    string
	ToCurrency      string
	OriginalAmount  decimal.Decimal
	ConvertedAmount decimal.Decimal
	Rate            decimal.Decimal
	InverseRate     decimal.Decimal
	Provider        string
}

// --- List Rates DTOs ---

// ListRatesRequest is the input DTO for listing exchange rates.
type ListRatesRequest struct {
	AsOf         time.Time
	BaseCurrency string
	TenantID     uuid.UUID
}

// ListRatesResponse is the output DTO for listing exchange rates.
type ListRatesResponse struct {
	Rates []ExchangeRateResponse
}

// --- Revaluation DTOs ---

// RevaluateRequest is the input DTO for running an FX revaluation.
type RevaluateRequest struct {
	FunctionalCurrency string
	Positions          []ForeignCurrencyPositionDTO
	TenantID           uuid.UUID
}

// ForeignCurrencyPositionDTO transfers a foreign-currency position.
type ForeignCurrencyPositionDTO struct {
	AccountCode string
	Currency    string
	Amount      decimal.Decimal
}

// RevaluateResponse is the output DTO for the revaluation result.
type RevaluateResponse struct {
	FunctionalCurrency string
	TotalGainLoss      decimal.Decimal
	Entries            []RevaluationEntryDTO
	TenantID           uuid.UUID
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
