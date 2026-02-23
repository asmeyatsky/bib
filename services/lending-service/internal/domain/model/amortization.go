package model

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// AmortizationEntry is an immutable value object representing one period in an
// amortization schedule.
type AmortizationEntry struct {
	DueDate          time.Time
	Principal        decimal.Decimal
	Interest         decimal.Decimal
	Total            decimal.Decimal
	RemainingBalance decimal.Decimal
	Period           int
}

// GenerateAmortizationSchedule computes a standard fixed-payment amortization
// schedule.
//
// Parameters:
//   - principal:     the loan amount
//   - annualRateBps: annual interest rate in basis points (e.g. 500 = 5.00%)
//   - termMonths:    number of monthly periods
//   - startDate:     the date from which the first payment is due (one month later)
//
// The calculation uses:
//
//	monthlyRate = annualRateBps / 10_000 / 12
//	payment     = P * r * (1+r)^n / ((1+r)^n - 1)
func GenerateAmortizationSchedule(
	principal decimal.Decimal,
	annualRateBps int,
	termMonths int,
	startDate time.Time,
) []AmortizationEntry {
	if termMonths <= 0 || principal.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	// Convert basis points to a monthly decimal rate using float64 for the
	// power calculation, then switch back to decimal for monetary arithmetic.
	annualRate := float64(annualRateBps) / 10_000.0
	monthlyRate := annualRate / 12.0

	n := float64(termMonths)
	var monthlyPayment decimal.Decimal

	if monthlyRate == 0 {
		// Zero-interest: even split.
		monthlyPayment = principal.Div(decimal.NewFromInt(int64(termMonths)))
	} else {
		// P * r * (1+r)^n / ((1+r)^n - 1)
		factor := math.Pow(1+monthlyRate, n)
		paymentFloat := principal.InexactFloat64() * monthlyRate * factor / (factor - 1)
		monthlyPayment = decimal.NewFromFloat(paymentFloat).Round(2)
	}

	schedule := make([]AmortizationEntry, 0, termMonths)
	remaining := principal
	monthlyRateDec := decimal.NewFromFloat(monthlyRate)

	for period := 1; period <= termMonths; period++ {
		dueDate := startDate.AddDate(0, period, 0)

		interest := remaining.Mul(monthlyRateDec).Round(2)
		principalPart := monthlyPayment.Sub(interest)

		// Last period: adjust for rounding so balance reaches exactly zero.
		if period == termMonths {
			principalPart = remaining
			interest = remaining.Mul(monthlyRateDec).Round(2)
			monthlyPayment = principalPart.Add(interest)
		}

		remaining = remaining.Sub(principalPart)
		if remaining.LessThan(decimal.Zero) {
			remaining = decimal.Zero
		}

		schedule = append(schedule, AmortizationEntry{
			Period:           period,
			DueDate:          dueDate,
			Principal:        principalPart,
			Interest:         interest,
			Total:            principalPart.Add(interest),
			RemainingBalance: remaining,
		})
	}

	return schedule
}
