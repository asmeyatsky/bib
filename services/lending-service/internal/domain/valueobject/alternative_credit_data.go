package valueobject

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// AlternativeDataType identifies the category of alternative credit data.
type AlternativeDataType string

const (
	AltDataUtility  AlternativeDataType = "UTILITY"
	AltDataRent     AlternativeDataType = "RENT"
	AltDataPayroll  AlternativeDataType = "PAYROLL"
	AltDataTelecom  AlternativeDataType = "TELECOM"
	AltDataInsurance AlternativeDataType = "INSURANCE"
)

// PaymentConsistency describes the regularity of payments.
type PaymentConsistency string

const (
	ConsistencyExcellent PaymentConsistency = "EXCELLENT" // 95%+ on time
	ConsistencyGood      PaymentConsistency = "GOOD"      // 85-94% on time
	ConsistencyFair      PaymentConsistency = "FAIR"      // 70-84% on time
	ConsistencyPoor      PaymentConsistency = "POOR"      // <70% on time
)

// AlternativeCreditRecord represents a single alternative credit data point
// used for scoring applicants who lack traditional credit history.
type AlternativeCreditRecord struct {
	dataType     AlternativeDataType
	provider     string          // e.g. "ConEd", "Verizon", "Landlord Inc"
	monthsOnFile int             // number of months with payment history
	onTimeRate   decimal.Decimal // percentage of on-time payments (0-100)
	avgMonthlyAmount decimal.Decimal
	lastPaymentDate  time.Time
	consistency  PaymentConsistency
}

// NewAlternativeCreditRecord creates a validated alternative credit record.
func NewAlternativeCreditRecord(
	dataType AlternativeDataType,
	provider string,
	monthsOnFile int,
	onTimeRate decimal.Decimal,
	avgMonthlyAmount decimal.Decimal,
	lastPaymentDate time.Time,
) (AlternativeCreditRecord, error) {
	if dataType == "" {
		return AlternativeCreditRecord{}, fmt.Errorf("data type is required")
	}
	if provider == "" {
		return AlternativeCreditRecord{}, fmt.Errorf("provider is required")
	}
	if monthsOnFile < 0 {
		return AlternativeCreditRecord{}, fmt.Errorf("months on file must not be negative")
	}
	if onTimeRate.LessThan(decimal.Zero) || onTimeRate.GreaterThan(decimal.NewFromInt(100)) {
		return AlternativeCreditRecord{}, fmt.Errorf("on-time rate must be between 0 and 100")
	}
	if avgMonthlyAmount.LessThan(decimal.Zero) {
		return AlternativeCreditRecord{}, fmt.Errorf("average monthly amount must not be negative")
	}

	consistency := deriveConsistency(onTimeRate)

	return AlternativeCreditRecord{
		dataType:         dataType,
		provider:         provider,
		monthsOnFile:     monthsOnFile,
		onTimeRate:       onTimeRate,
		avgMonthlyAmount: avgMonthlyAmount,
		lastPaymentDate:  lastPaymentDate,
		consistency:      consistency,
	}, nil
}

// DataType returns the category of alternative data.
func (r AlternativeCreditRecord) DataType() AlternativeDataType { return r.dataType }

// Provider returns the data provider name.
func (r AlternativeCreditRecord) Provider() string { return r.provider }

// MonthsOnFile returns the number of months of payment history.
func (r AlternativeCreditRecord) MonthsOnFile() int { return r.monthsOnFile }

// OnTimeRate returns the on-time payment percentage.
func (r AlternativeCreditRecord) OnTimeRate() decimal.Decimal { return r.onTimeRate }

// AvgMonthlyAmount returns the average monthly payment amount.
func (r AlternativeCreditRecord) AvgMonthlyAmount() decimal.Decimal { return r.avgMonthlyAmount }

// LastPaymentDate returns the date of the most recent payment.
func (r AlternativeCreditRecord) LastPaymentDate() time.Time { return r.lastPaymentDate }

// Consistency returns the derived payment consistency rating.
func (r AlternativeCreditRecord) Consistency() PaymentConsistency { return r.consistency }

// deriveConsistency maps on-time payment rate to a consistency rating.
func deriveConsistency(onTimeRate decimal.Decimal) PaymentConsistency {
	rate := onTimeRate.IntPart()
	switch {
	case rate >= 95:
		return ConsistencyExcellent
	case rate >= 85:
		return ConsistencyGood
	case rate >= 70:
		return ConsistencyFair
	default:
		return ConsistencyPoor
	}
}

// AlternativeCreditProfile aggregates multiple alternative credit records
// for a single applicant.
type AlternativeCreditProfile struct {
	ApplicantID string
	Records     []AlternativeCreditRecord
}

// TotalMonthsOfHistory returns the maximum months on file across all records.
func (p AlternativeCreditProfile) TotalMonthsOfHistory() int {
	max := 0
	for _, r := range p.Records {
		if r.MonthsOnFile() > max {
			max = r.MonthsOnFile()
		}
	}
	return max
}

// AverageOnTimeRate computes the weighted average on-time rate across records.
func (p AlternativeCreditProfile) AverageOnTimeRate() decimal.Decimal {
	if len(p.Records) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, r := range p.Records {
		sum = sum.Add(r.OnTimeRate())
	}
	return sum.Div(decimal.NewFromInt(int64(len(p.Records)))).Round(2)
}

// RecordCount returns the number of alternative data records.
func (p AlternativeCreditProfile) RecordCount() int {
	return len(p.Records)
}
