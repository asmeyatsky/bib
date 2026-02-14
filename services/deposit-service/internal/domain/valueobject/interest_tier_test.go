package valueobject_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

func TestNewInterestTier_Valid(t *testing.T) {
	minBal := decimal.NewFromInt(0)
	maxBal := decimal.NewFromInt(10000)
	rateBps := 250

	tier, err := valueobject.NewInterestTier(minBal, maxBal, rateBps)
	require.NoError(t, err)

	assert.True(t, tier.MinBalance().Equal(minBal))
	assert.True(t, tier.MaxBalance().Equal(maxBal))
	assert.Equal(t, 250, tier.RateBps())
}

func TestNewInterestTier_MinGreaterThanMax(t *testing.T) {
	_, err := valueobject.NewInterestTier(
		decimal.NewFromInt(10000),
		decimal.NewFromInt(5000),
		250,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max balance must be greater than min balance")
}

func TestNewInterestTier_MinEqualsMax(t *testing.T) {
	_, err := valueobject.NewInterestTier(
		decimal.NewFromInt(10000),
		decimal.NewFromInt(10000),
		250,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max balance must be greater than min balance")
}

func TestNewInterestTier_NegativeMinBalance(t *testing.T) {
	_, err := valueobject.NewInterestTier(
		decimal.NewFromInt(-1),
		decimal.NewFromInt(10000),
		250,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min balance must not be negative")
}

func TestNewInterestTier_NegativeRate(t *testing.T) {
	_, err := valueobject.NewInterestTier(
		decimal.NewFromInt(0),
		decimal.NewFromInt(10000),
		-50,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate basis points must not be negative")
}

func TestNewInterestTier_ZeroRate(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(0),
		decimal.NewFromInt(10000),
		0,
	)
	require.NoError(t, err)
	assert.Equal(t, 0, tier.RateBps())
	assert.True(t, tier.AnnualRate().IsZero())
	assert.True(t, tier.DailyRate().IsZero())
}

func TestInterestTier_AnnualRate(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(0),
		decimal.NewFromInt(100000),
		250,
	)
	require.NoError(t, err)

	// 250 bps = 0.025
	expected := decimal.NewFromFloat(0.025)
	assert.True(t, tier.AnnualRate().Equal(expected),
		"expected %s, got %s", expected, tier.AnnualRate())
}

func TestInterestTier_AnnualRate_500Bps(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(0),
		decimal.NewFromInt(100000),
		500,
	)
	require.NoError(t, err)

	// 500 bps = 0.05
	expected := decimal.NewFromFloat(0.05)
	assert.True(t, tier.AnnualRate().Equal(expected),
		"expected %s, got %s", expected, tier.AnnualRate())
}

func TestInterestTier_DailyRate(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(0),
		decimal.NewFromInt(100000),
		250,
	)
	require.NoError(t, err)

	// 250 bps = 0.025 / 365
	expected := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	assert.True(t, tier.DailyRate().Equal(expected),
		"expected %s, got %s", expected, tier.DailyRate())
}

func TestInterestTier_Applies_WithinRange(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(1000),
		decimal.NewFromInt(50000),
		250,
	)
	require.NoError(t, err)

	assert.True(t, tier.Applies(decimal.NewFromInt(1000)))  // at min boundary
	assert.True(t, tier.Applies(decimal.NewFromInt(25000))) // middle
	assert.True(t, tier.Applies(decimal.NewFromInt(50000))) // at max boundary
}

func TestInterestTier_Applies_OutsideRange(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromInt(1000),
		decimal.NewFromInt(50000),
		250,
	)
	require.NoError(t, err)

	assert.False(t, tier.Applies(decimal.NewFromInt(999)))   // below min
	assert.False(t, tier.Applies(decimal.NewFromInt(50001))) // above max
	assert.False(t, tier.Applies(decimal.NewFromInt(0)))     // zero
}

func TestInterestTier_Applies_AtBoundaries(t *testing.T) {
	tier, err := valueobject.NewInterestTier(
		decimal.NewFromFloat(0.01),
		decimal.NewFromFloat(9999.99),
		100,
	)
	require.NoError(t, err)

	assert.True(t, tier.Applies(decimal.NewFromFloat(0.01)))
	assert.True(t, tier.Applies(decimal.NewFromFloat(9999.99)))
	assert.False(t, tier.Applies(decimal.NewFromFloat(0.009)))
	assert.False(t, tier.Applies(decimal.NewFromFloat(10000.00)))
}
