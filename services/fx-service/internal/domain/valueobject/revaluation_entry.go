package valueobject

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// RevaluationEntry is an immutable value object representing a single line item
// in an ASC 830 FX revaluation run. It captures the gain or loss from translating
// a foreign-currency balance into the functional currency at the current spot rate.
type RevaluationEntry struct {
	accountCode        string
	originalCurrency   string
	functionalCurrency string
	originalAmount     decimal.Decimal
	revaluedAmount     decimal.Decimal
	gainLoss           decimal.Decimal
	rate               SpotRate
}

// NewRevaluationEntry creates a RevaluationEntry with the provided values.
func NewRevaluationEntry(
	accountCode string,
	originalCurrency string,
	functionalCurrency string,
	originalAmount decimal.Decimal,
	revaluedAmount decimal.Decimal,
	gainLoss decimal.Decimal,
	rate SpotRate,
) RevaluationEntry {
	return RevaluationEntry{
		accountCode:        accountCode,
		originalCurrency:   originalCurrency,
		functionalCurrency: functionalCurrency,
		originalAmount:     originalAmount,
		revaluedAmount:     revaluedAmount,
		gainLoss:           gainLoss,
		rate:               rate,
	}
}

// AccountCode returns the GL account code for this entry.
func (re RevaluationEntry) AccountCode() string {
	return re.accountCode
}

// OriginalCurrency returns the ISO 4217 currency code of the original balance.
func (re RevaluationEntry) OriginalCurrency() string {
	return re.originalCurrency
}

// FunctionalCurrency returns the ISO 4217 code of the functional (reporting) currency.
func (re RevaluationEntry) FunctionalCurrency() string {
	return re.functionalCurrency
}

// OriginalAmount returns the balance in the original (foreign) currency.
func (re RevaluationEntry) OriginalAmount() decimal.Decimal {
	return re.originalAmount
}

// RevaluedAmount returns the balance converted to the functional currency.
func (re RevaluationEntry) RevaluedAmount() decimal.Decimal {
	return re.revaluedAmount
}

// GainLoss returns the unrealised gain or loss from the revaluation.
func (re RevaluationEntry) GainLoss() decimal.Decimal {
	return re.gainLoss
}

// Rate returns the spot rate used for the revaluation.
func (re RevaluationEntry) Rate() SpotRate {
	return re.rate
}

// String returns a human-readable description of the revaluation entry.
func (re RevaluationEntry) String() string {
	return fmt.Sprintf(
		"RevaluationEntry{account=%s, %s %s -> %s %s, gain/loss=%s, rate=%s}",
		re.accountCode,
		re.originalAmount.StringFixed(4), re.originalCurrency,
		re.revaluedAmount.StringFixed(4), re.functionalCurrency,
		re.gainLoss.StringFixed(4),
		re.rate.String(),
	)
}
