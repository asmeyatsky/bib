package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

func TestNewCurrencyPair_Valid(t *testing.T) {
	pair, err := valueobject.NewCurrencyPair("USD", "EUR")

	require.NoError(t, err)
	assert.Equal(t, "USD", pair.Base())
	assert.Equal(t, "EUR", pair.Quote())
	assert.Equal(t, "USD/EUR", pair.String())
}

func TestNewCurrencyPair_InvalidBaseEmpty(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("", "EUR")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base currency")
}

func TestNewCurrencyPair_InvalidBaseLowercase(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("usd", "EUR")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base currency")
}

func TestNewCurrencyPair_InvalidBaseTooLong(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("USDD", "EUR")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base currency")
}

func TestNewCurrencyPair_InvalidBaseTooShort(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("US", "EUR")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base currency")
}

func TestNewCurrencyPair_InvalidQuote(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("USD", "123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quote currency")
}

func TestNewCurrencyPair_SameCurrency(t *testing.T) {
	_, err := valueobject.NewCurrencyPair("USD", "USD")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must differ")
}

func TestCurrencyPair_Inverse(t *testing.T) {
	pair, err := valueobject.NewCurrencyPair("USD", "EUR")
	require.NoError(t, err)

	inv := pair.Inverse()

	assert.Equal(t, "EUR", inv.Base())
	assert.Equal(t, "USD", inv.Quote())
	assert.Equal(t, "EUR/USD", inv.String())
}

func TestCurrencyPair_Equal(t *testing.T) {
	pair1, _ := valueobject.NewCurrencyPair("USD", "EUR")
	pair2, _ := valueobject.NewCurrencyPair("USD", "EUR")
	pair3, _ := valueobject.NewCurrencyPair("EUR", "USD")

	assert.True(t, pair1.Equal(pair2))
	assert.False(t, pair1.Equal(pair3))
}

func TestCurrencyPair_InverseOfInverse(t *testing.T) {
	pair, _ := valueobject.NewCurrencyPair("GBP", "JPY")
	doubleInverse := pair.Inverse().Inverse()

	assert.True(t, pair.Equal(doubleInverse))
}
