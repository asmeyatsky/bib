package valueobject

import (
	"fmt"
	"regexp"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// CurrencyPair is an immutable value object representing a base/quote currency pair
// following ISO 4217 conventions (e.g., USD/EUR).
type CurrencyPair struct {
	base  string
	quote string
}

// NewCurrencyPair creates a CurrencyPair after validating both currencies are 3-letter
// uppercase ISO 4217 codes and that they differ from each other.
func NewCurrencyPair(base, quote string) (CurrencyPair, error) {
	if !currencyCodePattern.MatchString(base) {
		return CurrencyPair{}, fmt.Errorf("invalid base currency %q: must be exactly 3 uppercase letters", base)
	}
	if !currencyCodePattern.MatchString(quote) {
		return CurrencyPair{}, fmt.Errorf("invalid quote currency %q: must be exactly 3 uppercase letters", quote)
	}
	if base == quote {
		return CurrencyPair{}, fmt.Errorf("base and quote currencies must differ: %s/%s", base, quote)
	}
	return CurrencyPair{base: base, quote: quote}, nil
}

// Base returns the base currency code.
func (cp CurrencyPair) Base() string {
	return cp.base
}

// Quote returns the quote currency code.
func (cp CurrencyPair) Quote() string {
	return cp.quote
}

// String returns the pair formatted as "BASE/QUOTE" (e.g., "USD/EUR").
func (cp CurrencyPair) String() string {
	return fmt.Sprintf("%s/%s", cp.base, cp.quote)
}

// Inverse returns the inverted pair (e.g., USD/EUR becomes EUR/USD).
func (cp CurrencyPair) Inverse() CurrencyPair {
	return CurrencyPair{base: cp.quote, quote: cp.base}
}

// Equal returns true if both pairs have the same base and quote.
func (cp CurrencyPair) Equal(other CurrencyPair) bool {
	return cp.base == other.base && cp.quote == other.quote
}
