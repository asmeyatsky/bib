package money

import (
	"fmt"
	"regexp"

	"github.com/shopspring/decimal"
)

var currencyCodeRe = regexp.MustCompile(`^[A-Z]{3}$`)

// Currency is an ISO 4217 currency code.
type Currency struct {
	code string
}

// NewCurrency creates a Currency after validating the code is exactly 3 uppercase letters.
func NewCurrency(code string) (Currency, error) {
	if !currencyCodeRe.MatchString(code) {
		return Currency{}, fmt.Errorf("invalid currency code %q: must be exactly 3 uppercase letters", code)
	}
	return Currency{code: code}, nil
}

// MustCurrency creates a Currency and panics on error. Intended for package-level variable
// initialization only.
func MustCurrency(code string) Currency {
	c, err := NewCurrency(code)
	if err != nil {
		panic(err)
	}
	return c
}

// Code returns the ISO 4217 currency code.
func (c Currency) Code() string {
	return c.code
}

// String returns the currency code.
func (c Currency) String() string {
	return c.code
}

// Common currencies.
var (
	USD = MustCurrency("USD")
	EUR = MustCurrency("EUR")
	GBP = MustCurrency("GBP")
)

// Money represents an immutable monetary amount with currency.
// Fields are unexported to enforce immutability.
type Money struct {
	amount   decimal.Decimal
	currency Currency
}

// New creates a Money value from a decimal amount and currency.
func New(amount decimal.Decimal, currency Currency) Money {
	return Money{amount: amount, currency: currency}
}

// NewFromString parses an amount string and currency code into a Money value.
func NewFromString(amount string, currency string) (Money, error) {
	cur, err := NewCurrency(currency)
	if err != nil {
		return Money{}, fmt.Errorf("invalid currency: %w", err)
	}

	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount %q: %w", amount, err)
	}

	return Money{amount: d, currency: cur}, nil
}

// Zero returns a Money value of zero in the given currency.
func Zero(currency Currency) Money {
	return Money{amount: decimal.Zero, currency: currency}
}

// Amount returns the decimal amount.
func (m Money) Amount() decimal.Decimal {
	return m.amount
}

// Currency returns the currency.
func (m Money) Currency() Currency {
	return m.currency
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.amount.IsZero()
}

// IsPositive returns true if the amount is strictly greater than zero.
func (m Money) IsPositive() bool {
	return m.amount.IsPositive()
}

// IsNegative returns true if the amount is strictly less than zero.
func (m Money) IsNegative() bool {
	return m.amount.IsNegative()
}

// Add returns the sum of m and other. Returns an error if the currencies do not match.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: cannot add %s to %s", other.currency, m.currency)
	}
	return Money{amount: m.amount.Add(other.amount), currency: m.currency}, nil
}

// Subtract returns the difference of m minus other. Returns an error if the currencies do not match.
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: cannot subtract %s from %s", other.currency, m.currency)
	}
	return Money{amount: m.amount.Sub(other.amount), currency: m.currency}, nil
}

// Multiply returns m multiplied by the given factor.
func (m Money) Multiply(factor decimal.Decimal) Money {
	return Money{amount: m.amount.Mul(factor), currency: m.currency}
}

// Negate returns m with the sign of the amount flipped.
func (m Money) Negate() Money {
	return Money{amount: m.amount.Neg(), currency: m.currency}
}

// Abs returns m with the absolute value of the amount.
func (m Money) Abs() Money {
	return Money{amount: m.amount.Abs(), currency: m.currency}
}

// Equal returns true if both the amount and currency of m and other are equal.
func (m Money) Equal(other Money) bool {
	return m.currency == other.currency && m.amount.Equal(other.amount)
}

// String formats the Money value as "<amount> <currency>", for example "100.0000 USD".
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.amount.StringFixed(4), m.currency.Code())
}
