package service

import (
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// ForeignCurrencyPosition represents a balance held in a foreign currency
// that needs to be revalued into the functional currency per ASC 830.
type ForeignCurrencyPosition struct {
	AccountCode string
	Currency    string
	Amount      decimal.Decimal
}

// RevaluationEngine is a domain service that performs FX revaluation
// in accordance with ASC 830 (Foreign Currency Matters).
//
// For each foreign-currency position it:
//  1. Looks up the current spot rate for that currency against the functional currency.
//  2. Converts the position's amount to the functional currency.
//  3. Computes the unrealised gain/loss as the difference between the revalued
//     amount and the original amount (which represents the previously recorded
//     functional-currency equivalent).
type RevaluationEngine struct{}

// NewRevaluationEngine creates a new RevaluationEngine.
func NewRevaluationEngine() *RevaluationEngine {
	return &RevaluationEngine{}
}

// Revaluate performs the revaluation calculation for a set of foreign-currency
// positions. It returns the individual revaluation entries and the total
// aggregate gain/loss across all positions.
//
// The rates map is keyed by the foreign currency code (e.g., "EUR") and contains
// the spot rate expressing how much 1 unit of that currency is worth in the
// functional currency.
//
// Positions whose currency matches the functional currency or whose currency
// has no rate in the map are silently skipped.
func (re *RevaluationEngine) Revaluate(
	positions []ForeignCurrencyPosition,
	rates map[string]valueobject.SpotRate,
	functionalCurrency string,
) ([]valueobject.RevaluationEntry, decimal.Decimal) {
	var entries []valueobject.RevaluationEntry
	totalGainLoss := decimal.Zero

	for _, pos := range positions {
		// Skip positions already denominated in the functional currency.
		if pos.Currency == functionalCurrency {
			continue
		}

		rate, ok := rates[pos.Currency]
		if !ok {
			continue
		}

		revaluedAmount := rate.Convert(pos.Amount)
		// The gain/loss is the difference between the newly revalued amount
		// and the original amount (the previously booked functional-currency value).
		gainLoss := revaluedAmount.Sub(pos.Amount)

		entry := valueobject.NewRevaluationEntry(
			pos.AccountCode,
			pos.Currency,
			functionalCurrency,
			pos.Amount,
			revaluedAmount,
			gainLoss,
			rate,
		)

		entries = append(entries, entry)
		totalGainLoss = totalGainLoss.Add(gainLoss)
	}

	return entries, totalGainLoss
}
