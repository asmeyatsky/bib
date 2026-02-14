package valueobject_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

func TestNewSpotRate_Valid(t *testing.T) {
	rate, err := valueobject.NewSpotRate(decimal.NewFromFloat(1.2345))

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(1.2345).Equal(rate.Rate()))
}

func TestNewSpotRate_Zero(t *testing.T) {
	_, err := valueobject.NewSpotRate(decimal.Zero)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestNewSpotRate_Negative(t *testing.T) {
	_, err := valueobject.NewSpotRate(decimal.NewFromFloat(-1.5))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestSpotRate_Inverse(t *testing.T) {
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(2.0))

	inv := rate.Inverse()

	expected := decimal.NewFromFloat(0.5)
	assert.True(t, expected.Equal(inv.Rate()), "expected %s, got %s", expected, inv.Rate())
}

func TestSpotRate_Convert(t *testing.T) {
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.25))
	amount := decimal.NewFromFloat(100.0)

	converted := rate.Convert(amount)

	expected := decimal.NewFromFloat(125.0)
	assert.True(t, expected.Equal(converted), "expected %s, got %s", expected, converted)
}

func TestSpotRate_Convert_SmallAmount(t *testing.T) {
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.85))
	amount := decimal.NewFromFloat(1.0)

	converted := rate.Convert(amount)

	expected := decimal.NewFromFloat(0.85)
	assert.True(t, expected.Equal(converted))
}

func TestSpotRate_String(t *testing.T) {
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.2345))

	str := rate.String()

	assert.Equal(t, "1.2345000000", str)
}

func TestSpotRate_Equal(t *testing.T) {
	r1, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.5))
	r2, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.5))
	r3, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.6))

	assert.True(t, r1.Equal(r2))
	assert.False(t, r1.Equal(r3))
}

func TestSpotRate_IsZero(t *testing.T) {
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.0))

	assert.False(t, rate.IsZero())
	assert.True(t, valueobject.SpotRate{}.IsZero())
}
