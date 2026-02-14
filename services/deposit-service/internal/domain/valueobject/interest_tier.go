package valueobject

import (
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	bpsBase  = decimal.NewFromInt(10000)
	daysYear = decimal.NewFromInt(365)
)

// InterestTier is an immutable value object representing a balance-based interest rate tier.
// Rates are expressed in basis points (bps): 250 bps = 2.50%.
type InterestTier struct {
	minBalance decimal.Decimal
	maxBalance decimal.Decimal
	rateBps    int
}

// NewInterestTier creates a validated InterestTier. It enforces that min < max and rate >= 0.
func NewInterestTier(minBalance, maxBalance decimal.Decimal, rateBps int) (InterestTier, error) {
	if minBalance.IsNegative() {
		return InterestTier{}, fmt.Errorf("min balance must not be negative")
	}
	if maxBalance.LessThanOrEqual(minBalance) {
		return InterestTier{}, fmt.Errorf("max balance must be greater than min balance")
	}
	if rateBps < 0 {
		return InterestTier{}, fmt.Errorf("rate basis points must not be negative")
	}
	return InterestTier{
		minBalance: minBalance,
		maxBalance: maxBalance,
		rateBps:    rateBps,
	}, nil
}

// MinBalance returns the lower bound of the tier (inclusive).
func (t InterestTier) MinBalance() decimal.Decimal {
	return t.minBalance
}

// MaxBalance returns the upper bound of the tier (inclusive).
func (t InterestTier) MaxBalance() decimal.Decimal {
	return t.maxBalance
}

// RateBps returns the interest rate in basis points.
func (t InterestTier) RateBps() int {
	return t.rateBps
}

// AnnualRate returns the annual interest rate as a decimal (e.g. 250 bps -> 0.025).
func (t InterestTier) AnnualRate() decimal.Decimal {
	return decimal.NewFromInt(int64(t.rateBps)).Div(bpsBase)
}

// DailyRate returns the daily interest rate (annual rate / 365).
func (t InterestTier) DailyRate() decimal.Decimal {
	return t.AnnualRate().Div(daysYear)
}

// Applies returns true if the given balance falls within this tier's range [min, max].
func (t InterestTier) Applies(balance decimal.Decimal) bool {
	return balance.GreaterThanOrEqual(t.minBalance) && balance.LessThanOrEqual(t.maxBalance)
}
