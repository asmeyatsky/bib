package tests

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/bibbank/bib/services/card-service/internal/domain/service"
)

func TestJITFundingService_SufficientFunds(t *testing.T) {
	svc := service.NewJITFundingService()

	tests := []struct {
		name             string
		availableBalance decimal.Decimal
		transactionAmt   decimal.Decimal
	}{
		{
			name:             "exact amount",
			availableBalance: decimal.NewFromInt(100),
			transactionAmt:   decimal.NewFromInt(100),
		},
		{
			name:             "balance exceeds amount",
			availableBalance: decimal.NewFromInt(1000),
			transactionAmt:   decimal.NewFromInt(50),
		},
		{
			name:             "fractional amounts",
			availableBalance: decimal.NewFromFloat(100.50),
			transactionAmt:   decimal.NewFromFloat(100.49),
		},
		{
			name:             "large amounts",
			availableBalance: decimal.NewFromInt(999999),
			transactionAmt:   decimal.NewFromInt(500000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CheckFunding(tt.availableBalance, tt.transactionAmt)
			assert.True(t, result.Approved, "expected funding to be approved")
			assert.Empty(t, result.DeclineReason, "expected no decline reason")
		})
	}
}

func TestJITFundingService_InsufficientFunds(t *testing.T) {
	svc := service.NewJITFundingService()

	tests := []struct {
		name             string
		availableBalance decimal.Decimal
		transactionAmt   decimal.Decimal
		expectedReason   string
	}{
		{
			name:             "zero balance",
			availableBalance: decimal.Zero,
			transactionAmt:   decimal.NewFromInt(100),
			expectedReason:   "insufficient funds",
		},
		{
			name:             "balance slightly less than amount",
			availableBalance: decimal.NewFromFloat(99.99),
			transactionAmt:   decimal.NewFromInt(100),
			expectedReason:   "insufficient funds",
		},
		{
			name:             "negative balance",
			availableBalance: decimal.NewFromInt(-50),
			transactionAmt:   decimal.NewFromInt(10),
			expectedReason:   "insufficient funds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CheckFunding(tt.availableBalance, tt.transactionAmt)
			assert.False(t, result.Approved, "expected funding to be declined")
			assert.Equal(t, tt.expectedReason, result.DeclineReason)
		})
	}
}

func TestJITFundingService_InvalidAmount(t *testing.T) {
	svc := service.NewJITFundingService()

	tests := []struct {
		name           string
		transactionAmt decimal.Decimal
		expectedReason string
	}{
		{
			name:           "zero transaction amount",
			transactionAmt: decimal.Zero,
			expectedReason: "transaction amount must be positive",
		},
		{
			name:           "negative transaction amount",
			transactionAmt: decimal.NewFromInt(-50),
			expectedReason: "transaction amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CheckFunding(decimal.NewFromInt(1000), tt.transactionAmt)
			assert.False(t, result.Approved, "expected funding to be declined")
			assert.Equal(t, tt.expectedReason, result.DeclineReason)
		})
	}
}
