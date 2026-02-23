package service

import (
	"strconv"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// UnderwritingEngine – domain service for AI-driven underwriting rules
// ---------------------------------------------------------------------------

// UnderwritingResult holds the outcome of the underwriting evaluation.
type UnderwritingResult struct {
	Reason        string
	CreditScore   string
	MaxAmount     decimal.Decimal
	SuggestedRate int
	Approved      bool
}

// UnderwritingEngine encapsulates rule-based credit decisioning.
type UnderwritingEngine struct{}

// NewUnderwritingEngine returns a new engine instance.
func NewUnderwritingEngine() *UnderwritingEngine {
	return &UnderwritingEngine{}
}

// Evaluate performs a simplified, rule-based underwriting decision.
//
// Tiers:
//
//	score >= 750  -> approved, max $500K, 450 bps
//	score >= 700  -> approved, max $250K, 550 bps
//	score >= 600  -> approved, max $100K, 850 bps (higher rate)
//	score <  600  -> rejected
func (e *UnderwritingEngine) Evaluate(
	creditScore string,
	requestedAmount decimal.Decimal,
	termMonths int,
) UnderwritingResult {
	score, err := strconv.Atoi(creditScore)
	if err != nil {
		return UnderwritingResult{
			Approved:    false,
			Reason:      "unable to parse credit score",
			CreditScore: creditScore,
		}
	}

	var (
		approved      bool
		reason        string
		maxAmount     decimal.Decimal
		suggestedRate int
	)

	switch {
	case score >= 750:
		approved = true
		reason = "excellent credit tier"
		maxAmount = decimal.NewFromInt(500_000)
		suggestedRate = 450
	case score >= 700:
		approved = true
		reason = "good credit tier"
		maxAmount = decimal.NewFromInt(250_000)
		suggestedRate = 550
	case score >= 600:
		approved = true
		reason = "fair credit tier - elevated rate applies"
		maxAmount = decimal.NewFromInt(100_000)
		suggestedRate = 850
	default:
		approved = false
		reason = "credit score below minimum threshold"
		maxAmount = decimal.Zero
		suggestedRate = 0
	}

	// If approved but requested amount exceeds the tier limit, reject.
	if approved && requestedAmount.GreaterThan(maxAmount) {
		approved = false
		reason = "requested amount exceeds maximum for credit tier"
	}

	// Term sanity check.
	if approved && termMonths > 360 {
		approved = false
		reason = "term exceeds maximum 360 months"
	}

	return UnderwritingResult{
		Approved:      approved,
		Reason:        reason,
		CreditScore:   creditScore,
		MaxAmount:     maxAmount,
		SuggestedRate: suggestedRate,
	}
}
