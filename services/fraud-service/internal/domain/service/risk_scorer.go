package service

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RiskInput contains the data required for risk scoring.
type RiskInput struct {
	Amount          decimal.Decimal
	Currency        string
	AccountID       uuid.UUID
	TransactionType string
	Metadata        map[string]string
}

// RiskOutput contains the result of risk scoring.
type RiskOutput struct {
	Score   int
	Signals []string
}

// RiskScorer is a domain service that calculates risk scores using rule-based logic.
type RiskScorer struct{}

// NewRiskScorer creates a new RiskScorer instance.
func NewRiskScorer() *RiskScorer {
	return &RiskScorer{}
}

// Score evaluates the risk of a transaction based on rule-based heuristics.
// The base score is 10. Various rules add points and corresponding signals.
func (s *RiskScorer) Score(input RiskInput) RiskOutput {
	score := 10
	signals := make([]string, 0)

	// Rule: High-value transaction (amount > 10,000).
	highValueThreshold := decimal.NewFromInt(10000)
	if input.Amount.GreaterThan(highValueThreshold) {
		score += 20
		signals = append(signals, "high_value")
	}

	// Rule: Very high-value transaction (amount > 50,000).
	veryHighValueThreshold := decimal.NewFromInt(50000)
	if input.Amount.GreaterThan(veryHighValueThreshold) {
		score += 15
		signals = append(signals, "very_high_value")
	}

	// Rule: International / cross-border transaction.
	if input.Metadata != nil {
		if country, ok := input.Metadata["destination_country"]; ok && country != "" {
			if sourceCountry, sok := input.Metadata["source_country"]; sok && sourceCountry != country {
				score += 15
				signals = append(signals, "cross_border")
			}
		}
	}

	// Rule: High-risk transaction types.
	switch input.TransactionType {
	case "wire_transfer":
		score += 10
		signals = append(signals, "wire_transfer")
	case "crypto_purchase":
		score += 20
		signals = append(signals, "crypto_transaction")
	case "cash_withdrawal":
		score += 5
		signals = append(signals, "cash_withdrawal")
	}

	// Rule: New account (flagged via metadata).
	if input.Metadata != nil {
		if val, ok := input.Metadata["account_age"]; ok && val == "new" {
			score += 10
			signals = append(signals, "new_account")
		}
	}

	// Rule: Unusual currency.
	unusualCurrencies := map[string]bool{
		"XMR": true, "BTC": true, "ETH": true,
	}
	if unusualCurrencies[input.Currency] {
		score += 10
		signals = append(signals, "unusual_currency")
	}

	// Rule: High-risk country.
	if input.Metadata != nil {
		if country, ok := input.Metadata["destination_country"]; ok {
			highRiskCountries := map[string]bool{
				"KP": true, "IR": true, "SY": true, "CU": true,
			}
			if highRiskCountries[country] {
				score += 25
				signals = append(signals, "high_risk_country")
			}
		}
	}

	// Rule: Rapid successive transactions.
	if input.Metadata != nil {
		if val, ok := input.Metadata["rapid_transactions"]; ok && val == "true" {
			score += 15
			signals = append(signals, "rapid_transactions")
		}
	}

	// Cap score at 100.
	if score > 100 {
		score = 100
	}

	return RiskOutput{
		Score:   score,
		Signals: signals,
	}
}
