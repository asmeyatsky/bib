package valueobject

import (
	"fmt"
	"time"
)

// FiscalPeriod represents a fiscal/accounting period (year + month).
type FiscalPeriod struct {
	year  int
	month time.Month
}

func NewFiscalPeriod(year int, month time.Month) (FiscalPeriod, error) {
	if year < 2000 || year > 2100 {
		return FiscalPeriod{}, fmt.Errorf("invalid fiscal year %d: must be between 2000 and 2100", year)
	}
	if month < time.January || month > time.December {
		return FiscalPeriod{}, fmt.Errorf("invalid month %d", month)
	}
	return FiscalPeriod{year: year, month: month}, nil
}

func FiscalPeriodFromTime(t time.Time) FiscalPeriod {
	return FiscalPeriod{year: t.Year(), month: t.Month()}
}

func (fp FiscalPeriod) Year() int         { return fp.year }
func (fp FiscalPeriod) Month() time.Month { return fp.month }
func (fp FiscalPeriod) IsZero() bool      { return fp.year == 0 }

func (fp FiscalPeriod) String() string {
	return fmt.Sprintf("%d-%02d", fp.year, fp.month)
}

func (fp FiscalPeriod) StartDate() time.Time {
	return time.Date(fp.year, fp.month, 1, 0, 0, 0, 0, time.UTC)
}

func (fp FiscalPeriod) EndDate() time.Time {
	return fp.StartDate().AddDate(0, 1, -1)
}

func (fp FiscalPeriod) Contains(t time.Time) bool {
	return t.Year() == fp.year && t.Month() == fp.month
}

func (fp FiscalPeriod) Next() FiscalPeriod {
	if fp.month == time.December {
		return FiscalPeriod{year: fp.year + 1, month: time.January}
	}
	return FiscalPeriod{year: fp.year, month: fp.month + 1}
}

func (fp FiscalPeriod) Previous() FiscalPeriod {
	if fp.month == time.January {
		return FiscalPeriod{year: fp.year - 1, month: time.December}
	}
	return FiscalPeriod{year: fp.year, month: fp.month - 1}
}

// PeriodStatus tracks whether a fiscal period is open or closed.
type PeriodStatus string

const (
	PeriodStatusOpen   PeriodStatus = "OPEN"
	PeriodStatusClosed PeriodStatus = "CLOSED"
)
