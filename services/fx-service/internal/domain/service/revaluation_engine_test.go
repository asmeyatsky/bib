package service_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fx-service/internal/domain/service"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

func mustRate(t *testing.T, f float64) valueobject.SpotRate {
	t.Helper()
	r, err := valueobject.NewSpotRate(decimal.NewFromFloat(f))
	require.NoError(t, err)
	return r
}

func TestRevaluationEngine_SinglePosition(t *testing.T) {
	engine := service.NewRevaluationEngine()

	positions := []service.ForeignCurrencyPosition{
		{AccountCode: "1100", Currency: "EUR", Amount: decimal.NewFromFloat(1000.0)},
	}
	rates := map[string]valueobject.SpotRate{
		"EUR": mustRate(t, 1.10), // 1 EUR = 1.10 USD
	}

	entries, totalGainLoss := engine.Revaluate(positions, rates, "USD")

	require.Len(t, entries, 1)
	assert.Equal(t, "1100", entries[0].AccountCode())
	assert.Equal(t, "EUR", entries[0].OriginalCurrency())
	assert.Equal(t, "USD", entries[0].FunctionalCurrency())

	// Original: 1000 EUR; Revalued: 1000 * 1.10 = 1100 USD; GainLoss: 1100 - 1000 = 100
	assert.True(t, decimal.NewFromFloat(1000.0).Equal(entries[0].OriginalAmount()))
	assert.True(t, decimal.NewFromFloat(1100.0).Equal(entries[0].RevaluedAmount()))
	assert.True(t, decimal.NewFromFloat(100.0).Equal(entries[0].GainLoss()))
	assert.True(t, decimal.NewFromFloat(100.0).Equal(totalGainLoss))
}

func TestRevaluationEngine_MultiplePositions(t *testing.T) {
	engine := service.NewRevaluationEngine()

	positions := []service.ForeignCurrencyPosition{
		{AccountCode: "1100", Currency: "EUR", Amount: decimal.NewFromFloat(1000.0)},
		{AccountCode: "1200", Currency: "GBP", Amount: decimal.NewFromFloat(500.0)},
	}
	rates := map[string]valueobject.SpotRate{
		"EUR": mustRate(t, 1.10),
		"GBP": mustRate(t, 1.30),
	}

	entries, totalGainLoss := engine.Revaluate(positions, rates, "USD")

	require.Len(t, entries, 2)

	// EUR position: revalued = 1100, gain = 100
	assert.True(t, decimal.NewFromFloat(100.0).Equal(entries[0].GainLoss()))
	// GBP position: revalued = 650, gain = 150
	assert.True(t, decimal.NewFromFloat(150.0).Equal(entries[1].GainLoss()))
	// Total: 100 + 150 = 250
	assert.True(t, decimal.NewFromFloat(250.0).Equal(totalGainLoss))
}

func TestRevaluationEngine_SkipsFunctionalCurrency(t *testing.T) {
	engine := service.NewRevaluationEngine()

	positions := []service.ForeignCurrencyPosition{
		{AccountCode: "1000", Currency: "USD", Amount: decimal.NewFromFloat(500.0)},
		{AccountCode: "1100", Currency: "EUR", Amount: decimal.NewFromFloat(1000.0)},
	}
	rates := map[string]valueobject.SpotRate{
		"EUR": mustRate(t, 1.10),
	}

	entries, _ := engine.Revaluate(positions, rates, "USD")

	// Only the EUR position should be included.
	require.Len(t, entries, 1)
	assert.Equal(t, "EUR", entries[0].OriginalCurrency())
}

func TestRevaluationEngine_SkipsMissingRate(t *testing.T) {
	engine := service.NewRevaluationEngine()

	positions := []service.ForeignCurrencyPosition{
		{AccountCode: "1100", Currency: "EUR", Amount: decimal.NewFromFloat(1000.0)},
		{AccountCode: "1200", Currency: "JPY", Amount: decimal.NewFromFloat(50000.0)},
	}
	rates := map[string]valueobject.SpotRate{
		"EUR": mustRate(t, 1.10),
		// No JPY rate provided.
	}

	entries, totalGainLoss := engine.Revaluate(positions, rates, "USD")

	// Only EUR processed.
	require.Len(t, entries, 1)
	assert.Equal(t, "EUR", entries[0].OriginalCurrency())
	assert.True(t, decimal.NewFromFloat(100.0).Equal(totalGainLoss))
}

func TestRevaluationEngine_EmptyPositions(t *testing.T) {
	engine := service.NewRevaluationEngine()

	entries, totalGainLoss := engine.Revaluate(nil, nil, "USD")

	assert.Nil(t, entries)
	assert.True(t, decimal.Zero.Equal(totalGainLoss))
}

func TestRevaluationEngine_NegativeGainLoss(t *testing.T) {
	engine := service.NewRevaluationEngine()

	positions := []service.ForeignCurrencyPosition{
		{AccountCode: "1100", Currency: "EUR", Amount: decimal.NewFromFloat(1000.0)},
	}
	rates := map[string]valueobject.SpotRate{
		"EUR": mustRate(t, 0.90), // EUR weakened: 1 EUR = 0.90 USD
	}

	entries, totalGainLoss := engine.Revaluate(positions, rates, "USD")

	require.Len(t, entries, 1)
	// Revalued: 1000 * 0.90 = 900; GainLoss: 900 - 1000 = -100
	assert.True(t, decimal.NewFromFloat(-100.0).Equal(entries[0].GainLoss()))
	assert.True(t, decimal.NewFromFloat(-100.0).Equal(totalGainLoss))
}
