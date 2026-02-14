package valueobject_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func TestNewFiscalPeriod_Valid(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month time.Month
	}{
		{name: "January 2024", year: 2024, month: time.January},
		{name: "December 2024", year: 2024, month: time.December},
		{name: "lower bound year", year: 2000, month: time.June},
		{name: "upper bound year", year: 2100, month: time.June},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp, err := valueobject.NewFiscalPeriod(tt.year, tt.month)
			require.NoError(t, err)
			assert.Equal(t, tt.year, fp.Year())
			assert.Equal(t, tt.month, fp.Month())
			assert.False(t, fp.IsZero())
		})
	}
}

func TestNewFiscalPeriod_InvalidYear(t *testing.T) {
	tests := []struct {
		name string
		year int
	}{
		{name: "too low", year: 1999},
		{name: "too high", year: 2101},
		{name: "zero", year: 0},
		{name: "negative", year: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobject.NewFiscalPeriod(tt.year, time.January)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid fiscal year")
		})
	}
}

func TestNewFiscalPeriod_InvalidMonth(t *testing.T) {
	_, err := valueobject.NewFiscalPeriod(2024, time.Month(0))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month")

	_, err = valueobject.NewFiscalPeriod(2024, time.Month(13))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month")
}

func TestFiscalPeriodFromTime(t *testing.T) {
	ts := time.Date(2024, time.March, 15, 10, 30, 0, 0, time.UTC)
	fp := valueobject.FiscalPeriodFromTime(ts)

	assert.Equal(t, 2024, fp.Year())
	assert.Equal(t, time.March, fp.Month())
}

func TestFiscalPeriod_String(t *testing.T) {
	fp, err := valueobject.NewFiscalPeriod(2024, time.March)
	require.NoError(t, err)
	assert.Equal(t, "2024-03", fp.String())

	fp2, err := valueobject.NewFiscalPeriod(2024, time.November)
	require.NoError(t, err)
	assert.Equal(t, "2024-11", fp2.String())
}

func TestFiscalPeriod_IsZero(t *testing.T) {
	var zeroFP valueobject.FiscalPeriod
	assert.True(t, zeroFP.IsZero())

	fp, err := valueobject.NewFiscalPeriod(2024, time.January)
	require.NoError(t, err)
	assert.False(t, fp.IsZero())
}

func TestFiscalPeriod_StartDate(t *testing.T) {
	fp, err := valueobject.NewFiscalPeriod(2024, time.March)
	require.NoError(t, err)

	start := fp.StartDate()
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, time.March, start.Month())
	assert.Equal(t, 1, start.Day())
	assert.Equal(t, 0, start.Hour())
}

func TestFiscalPeriod_EndDate(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    time.Month
		expected int // expected day of end date
	}{
		{name: "January has 31 days", year: 2024, month: time.January, expected: 31},
		{name: "February leap year", year: 2024, month: time.February, expected: 29},
		{name: "February non-leap year", year: 2023, month: time.February, expected: 28},
		{name: "April has 30 days", year: 2024, month: time.April, expected: 30},
		{name: "December has 31 days", year: 2024, month: time.December, expected: 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp, err := valueobject.NewFiscalPeriod(tt.year, tt.month)
			require.NoError(t, err)

			end := fp.EndDate()
			assert.Equal(t, tt.expected, end.Day())
			assert.Equal(t, tt.month, end.Month())
			assert.Equal(t, tt.year, end.Year())
		})
	}
}

func TestFiscalPeriod_Contains(t *testing.T) {
	fp, err := valueobject.NewFiscalPeriod(2024, time.March)
	require.NoError(t, err)

	// Within period
	assert.True(t, fp.Contains(time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)))
	assert.True(t, fp.Contains(time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC)))
	assert.True(t, fp.Contains(time.Date(2024, time.March, 31, 23, 59, 59, 0, time.UTC)))

	// Outside period
	assert.False(t, fp.Contains(time.Date(2024, time.February, 28, 0, 0, 0, 0, time.UTC)))
	assert.False(t, fp.Contains(time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)))
	assert.False(t, fp.Contains(time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC)))
}

func TestFiscalPeriod_Next(t *testing.T) {
	t.Run("regular month", func(t *testing.T) {
		fp, err := valueobject.NewFiscalPeriod(2024, time.March)
		require.NoError(t, err)

		next := fp.Next()
		assert.Equal(t, 2024, next.Year())
		assert.Equal(t, time.April, next.Month())
	})

	t.Run("December rolls to January next year", func(t *testing.T) {
		fp, err := valueobject.NewFiscalPeriod(2024, time.December)
		require.NoError(t, err)

		next := fp.Next()
		assert.Equal(t, 2025, next.Year())
		assert.Equal(t, time.January, next.Month())
	})
}

func TestFiscalPeriod_Previous(t *testing.T) {
	t.Run("regular month", func(t *testing.T) {
		fp, err := valueobject.NewFiscalPeriod(2024, time.March)
		require.NoError(t, err)

		prev := fp.Previous()
		assert.Equal(t, 2024, prev.Year())
		assert.Equal(t, time.February, prev.Month())
	})

	t.Run("January rolls to December previous year", func(t *testing.T) {
		fp, err := valueobject.NewFiscalPeriod(2024, time.January)
		require.NoError(t, err)

		prev := fp.Previous()
		assert.Equal(t, 2023, prev.Year())
		assert.Equal(t, time.December, prev.Month())
	})
}

func TestPeriodStatus_Constants(t *testing.T) {
	assert.Equal(t, valueobject.PeriodStatus("OPEN"), valueobject.PeriodStatusOpen)
	assert.Equal(t, valueobject.PeriodStatus("CLOSED"), valueobject.PeriodStatusClosed)
}
