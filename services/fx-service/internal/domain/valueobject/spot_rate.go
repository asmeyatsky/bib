package valueobject

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// SpotRate is an immutable value object representing an exchange rate.
// The rate must always be positive.
type SpotRate struct {
	rate decimal.Decimal
}

// NewSpotRate creates a SpotRate after validating the rate is positive.
func NewSpotRate(rate decimal.Decimal) (SpotRate, error) {
	if !rate.IsPositive() {
		return SpotRate{}, fmt.Errorf("spot rate must be positive, got %s", rate.String())
	}
	return SpotRate{rate: rate}, nil
}

// Rate returns the underlying decimal rate value.
func (sr SpotRate) Rate() decimal.Decimal {
	return sr.rate
}

// Inverse returns a new SpotRate that is the multiplicative inverse (1/rate).
func (sr SpotRate) Inverse() SpotRate {
	inv := decimal.NewFromInt(1).Div(sr.rate)
	return SpotRate{rate: inv}
}

// Convert multiplies the given amount by this rate and returns the result.
func (sr SpotRate) Convert(amount decimal.Decimal) decimal.Decimal {
	return amount.Mul(sr.rate)
}

// String returns the rate formatted to 10 decimal places.
func (sr SpotRate) String() string {
	return sr.rate.StringFixed(10)
}

// Equal returns true if both rates are numerically equal.
func (sr SpotRate) Equal(other SpotRate) bool {
	return sr.rate.Equal(other.rate)
}

// IsZero returns true if the rate is the zero value (uninitialised).
func (sr SpotRate) IsZero() bool {
	return sr.rate.IsZero()
}
