package tests

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
)

func TestGenerateAmortizationSchedule_30YearMortgage(t *testing.T) {
	// $100,000 at 5.00% (500 bps) for 360 months (30 years)
	principal := decimal.NewFromInt(100_000)
	annualRateBps := 500
	termMonths := 360
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	schedule := model.GenerateAmortizationSchedule(principal, annualRateBps, termMonths, startDate)

	require.Len(t, schedule, 360, "schedule should have 360 entries")

	// First entry checks.
	first := schedule[0]
	assert.Equal(t, 1, first.Period)
	assert.Equal(t, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), first.DueDate)

	// Monthly payment for $100K at 5% for 30 years is approximately $536.82.
	expectedPayment := decimal.NewFromFloat(536.82)
	assert.True(t,
		first.Total.Sub(expectedPayment).Abs().LessThan(decimal.NewFromFloat(0.02)),
		"first payment should be approximately $536.82, got %s", first.Total,
	)

	// First month interest = 100000 * 0.05/12 = ~$416.67
	expectedInterest := decimal.NewFromFloat(416.67)
	assert.True(t,
		first.Interest.Sub(expectedInterest).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"first interest should be approximately $416.67, got %s", first.Interest,
	)

	// First month principal portion.
	expectedPrincipal := expectedPayment.Sub(expectedInterest)
	assert.True(t,
		first.Principal.Sub(expectedPrincipal).Abs().LessThan(decimal.NewFromFloat(0.02)),
		"first principal should be approximately %s, got %s", expectedPrincipal, first.Principal,
	)

	// Last entry: remaining balance should be zero.
	last := schedule[len(schedule)-1]
	assert.Equal(t, 360, last.Period)
	assert.True(t, last.RemainingBalance.Equal(decimal.Zero),
		"final remaining balance should be zero, got %s", last.RemainingBalance,
	)

	// Sum of all principal payments should equal original principal.
	totalPrincipal := decimal.Zero
	for _, entry := range schedule {
		totalPrincipal = totalPrincipal.Add(entry.Principal)
	}
	assert.True(t,
		totalPrincipal.Sub(principal).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"total principal paid should equal original principal ($100,000), got %s", totalPrincipal,
	)
}

func TestGenerateAmortizationSchedule_ShortTerm(t *testing.T) {
	// $10,000 at 8% (800 bps) for 12 months
	principal := decimal.NewFromInt(10_000)
	schedule := model.GenerateAmortizationSchedule(principal, 800, 12,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))

	require.Len(t, schedule, 12)

	last := schedule[11]
	assert.Equal(t, 12, last.Period)
	assert.True(t, last.RemainingBalance.Equal(decimal.Zero))

	totalPrincipal := decimal.Zero
	for _, e := range schedule {
		totalPrincipal = totalPrincipal.Add(e.Principal)
	}
	assert.True(t,
		totalPrincipal.Sub(principal).Abs().LessThan(decimal.NewFromFloat(0.01)),
		"total principal should equal $10,000, got %s", totalPrincipal,
	)
}

func TestGenerateAmortizationSchedule_ZeroRate(t *testing.T) {
	principal := decimal.NewFromInt(12_000)
	schedule := model.GenerateAmortizationSchedule(principal, 0, 12,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	require.Len(t, schedule, 12)

	for _, e := range schedule {
		assert.True(t, e.Interest.Equal(decimal.Zero), "interest should be zero at 0% rate")
		assert.True(t, e.Principal.Equal(decimal.NewFromInt(1000)),
			"each payment should be $1000, got %s", e.Principal)
	}
}

func TestGenerateAmortizationSchedule_InvalidInputs(t *testing.T) {
	t.Run("zero term", func(t *testing.T) {
		sched := model.GenerateAmortizationSchedule(decimal.NewFromInt(1000), 500, 0, time.Now())
		assert.Nil(t, sched)
	})

	t.Run("zero principal", func(t *testing.T) {
		sched := model.GenerateAmortizationSchedule(decimal.Zero, 500, 12, time.Now())
		assert.Nil(t, sched)
	})

	t.Run("negative principal", func(t *testing.T) {
		sched := model.GenerateAmortizationSchedule(decimal.NewFromInt(-1000), 500, 12, time.Now())
		assert.Nil(t, sched)
	})
}
