package valueobject

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// PromotionalRate is an immutable value object representing a promotional
// interest rate bonus applied during a campaign period.
type PromotionalRate struct {
	bonusRateBps        int             // additional basis points on top of standard rate
	eligibilityCriteria string          // human-readable eligibility description
	minDeposit          decimal.Decimal // minimum deposit to qualify
	maxDeposit          decimal.Decimal // maximum deposit eligible for promotion
}

// NewPromotionalRate creates a validated PromotionalRate.
func NewPromotionalRate(
	bonusRateBps int,
	eligibilityCriteria string,
	minDeposit, maxDeposit decimal.Decimal,
) (PromotionalRate, error) {
	if bonusRateBps <= 0 {
		return PromotionalRate{}, fmt.Errorf("bonus rate must be positive, got %d bps", bonusRateBps)
	}
	if bonusRateBps > 5000 {
		return PromotionalRate{}, fmt.Errorf("bonus rate exceeds maximum 5000 bps")
	}
	if eligibilityCriteria == "" {
		return PromotionalRate{}, fmt.Errorf("eligibility criteria is required")
	}
	if minDeposit.IsNegative() {
		return PromotionalRate{}, fmt.Errorf("minimum deposit must not be negative")
	}
	if maxDeposit.LessThanOrEqual(decimal.Zero) {
		return PromotionalRate{}, fmt.Errorf("maximum deposit must be positive")
	}
	if maxDeposit.LessThanOrEqual(minDeposit) {
		return PromotionalRate{}, fmt.Errorf("maximum deposit must exceed minimum deposit")
	}

	return PromotionalRate{
		bonusRateBps:        bonusRateBps,
		eligibilityCriteria: eligibilityCriteria,
		minDeposit:          minDeposit,
		maxDeposit:          maxDeposit,
	}, nil
}

// BonusRateBps returns the bonus rate in basis points.
func (p PromotionalRate) BonusRateBps() int { return p.bonusRateBps }

// BonusAnnualRate returns the annual bonus rate as a decimal (e.g. 100 bps -> 0.01).
func (p PromotionalRate) BonusAnnualRate() decimal.Decimal {
	return decimal.NewFromInt(int64(p.bonusRateBps)).Div(decimal.NewFromInt(10000))
}

// BonusDailyRate returns the daily bonus rate.
func (p PromotionalRate) BonusDailyRate() decimal.Decimal {
	return p.BonusAnnualRate().Div(decimal.NewFromInt(365))
}

// EligibilityCriteria returns the human-readable criteria description.
func (p PromotionalRate) EligibilityCriteria() string { return p.eligibilityCriteria }

// MinDeposit returns the minimum deposit amount for eligibility.
func (p PromotionalRate) MinDeposit() decimal.Decimal { return p.minDeposit }

// MaxDeposit returns the maximum deposit amount eligible for promotion.
func (p PromotionalRate) MaxDeposit() decimal.Decimal { return p.maxDeposit }

// IsEligible returns true if the given deposit amount qualifies for the promotion.
func (p PromotionalRate) IsEligible(amount decimal.Decimal) bool {
	return amount.GreaterThanOrEqual(p.minDeposit) && amount.LessThanOrEqual(p.maxDeposit)
}
