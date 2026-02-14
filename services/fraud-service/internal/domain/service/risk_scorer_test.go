package service_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
)

func TestRiskScorer_BaseScore(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(100),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata:        nil,
	})

	assert.Equal(t, 10, output.Score)
	assert.Empty(t, output.Signals)
}

func TestRiskScorer_HighValue(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(15000),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
	})

	assert.Equal(t, 30, output.Score)
	assert.Contains(t, output.Signals, "high_value")
}

func TestRiskScorer_VeryHighValue(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(75000),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
	})

	// Base 10 + high_value 20 + very_high_value 15 = 45
	assert.Equal(t, 45, output.Score)
	assert.Contains(t, output.Signals, "high_value")
	assert.Contains(t, output.Signals, "very_high_value")
}

func TestRiskScorer_CrossBorder(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata: map[string]string{
			"source_country":      "US",
			"destination_country": "GB",
		},
	})

	// Base 10 + cross_border 15 = 25
	assert.Equal(t, 25, output.Score)
	assert.Contains(t, output.Signals, "cross_border")
}

func TestRiskScorer_WireTransfer(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "wire_transfer",
	})

	// Base 10 + wire_transfer 10 = 20
	assert.Equal(t, 20, output.Score)
	assert.Contains(t, output.Signals, "wire_transfer")
}

func TestRiskScorer_CryptoPurchase(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "crypto_purchase",
	})

	// Base 10 + crypto_transaction 20 = 30
	assert.Equal(t, 30, output.Score)
	assert.Contains(t, output.Signals, "crypto_transaction")
}

func TestRiskScorer_NewAccount(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata: map[string]string{
			"account_age": "new",
		},
	})

	// Base 10 + new_account 10 = 20
	assert.Equal(t, 20, output.Score)
	assert.Contains(t, output.Signals, "new_account")
}

func TestRiskScorer_UnusualCurrency(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "BTC",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
	})

	// Base 10 + unusual_currency 10 = 20
	assert.Equal(t, 20, output.Score)
	assert.Contains(t, output.Signals, "unusual_currency")
}

func TestRiskScorer_HighRiskCountry(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata: map[string]string{
			"source_country":      "US",
			"destination_country": "KP",
		},
	})

	// Base 10 + cross_border 15 + high_risk_country 25 = 50
	assert.Equal(t, 50, output.Score)
	assert.Contains(t, output.Signals, "cross_border")
	assert.Contains(t, output.Signals, "high_risk_country")
}

func TestRiskScorer_RapidTransactions(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata: map[string]string{
			"rapid_transactions": "true",
		},
	})

	// Base 10 + rapid_transactions 15 = 25
	assert.Equal(t, 25, output.Score)
	assert.Contains(t, output.Signals, "rapid_transactions")
}

func TestRiskScorer_CombinedHighRisk(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(75000),
		Currency:        "BTC",
		AccountID:       uuid.New(),
		TransactionType: "crypto_purchase",
		Metadata: map[string]string{
			"account_age":         "new",
			"source_country":      "US",
			"destination_country": "IR",
			"rapid_transactions":  "true",
		},
	})

	// Base 10 + high_value 20 + very_high_value 15 + cross_border 15 +
	// crypto_transaction 20 + new_account 10 + unusual_currency 10 +
	// high_risk_country 25 + rapid_transactions 15 = 140 -> capped at 100
	assert.Equal(t, 100, output.Score)
	assert.Contains(t, output.Signals, "high_value")
	assert.Contains(t, output.Signals, "very_high_value")
	assert.Contains(t, output.Signals, "cross_border")
	assert.Contains(t, output.Signals, "crypto_transaction")
	assert.Contains(t, output.Signals, "new_account")
	assert.Contains(t, output.Signals, "unusual_currency")
	assert.Contains(t, output.Signals, "high_risk_country")
	assert.Contains(t, output.Signals, "rapid_transactions")
}

func TestRiskScorer_ScoreCappedAt100(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(100000),
		Currency:        "XMR",
		AccountID:       uuid.New(),
		TransactionType: "crypto_purchase",
		Metadata: map[string]string{
			"account_age":         "new",
			"source_country":      "US",
			"destination_country": "KP",
			"rapid_transactions":  "true",
		},
	})

	assert.LessOrEqual(t, output.Score, 100)
}

func TestRiskScorer_SameCountryNotCrossBorder(t *testing.T) {
	scorer := service.NewRiskScorer()

	output := scorer.Score(service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		AccountID:       uuid.New(),
		TransactionType: "transfer",
		Metadata: map[string]string{
			"source_country":      "US",
			"destination_country": "US",
		},
	})

	// Base 10 only, same country is not cross-border.
	assert.Equal(t, 10, output.Score)
	assert.NotContains(t, output.Signals, "cross_border")
}
