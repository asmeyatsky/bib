package service

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

// AlternativeScoringResult holds the outcome of alternative credit scoring.
type AlternativeScoringResult struct {
	Confidence string
	Reason     string
	Factors    []ScoringFactor
	Score      int
	Eligible   bool
}

// ScoringFactor represents a single factor in the scoring model.
type ScoringFactor struct {
	Name   string
	Weight decimal.Decimal
	Impact string
	Score  int
}

// AlternativeScoring is a domain service that computes credit scores from
// alternative data sources (utility payments, rent, payroll, telecom).
// It is designed to serve applicants who lack traditional credit history.
type AlternativeScoring struct {
	// MinRecords is the minimum number of alternative data records required.
	MinRecords int
	// MinMonthsHistory is the minimum months of payment history required.
	MinMonthsHistory int
}

// NewAlternativeScoring creates a new alternative scoring service.
func NewAlternativeScoring() *AlternativeScoring {
	return &AlternativeScoring{
		MinRecords:       2,
		MinMonthsHistory: 6,
	}
}

// Score evaluates an applicant's alternative credit profile and returns
// a credit score in the 300-850 range.
//
// Scoring model weights:
//   - Payment consistency: 40%
//   - Payment history length: 25%
//   - Diversity of data sources: 20%
//   - Payment amounts stability: 15%
func (s *AlternativeScoring) Score(profile valueobject.AlternativeCreditProfile) (AlternativeScoringResult, error) {
	if profile.ApplicantID == "" {
		return AlternativeScoringResult{}, fmt.Errorf("applicant ID is required")
	}

	// Check minimum eligibility
	if profile.RecordCount() < s.MinRecords {
		return AlternativeScoringResult{
			Eligible: false,
			Reason:   fmt.Sprintf("insufficient data: need at least %d records, have %d", s.MinRecords, profile.RecordCount()),
		}, nil
	}
	if profile.TotalMonthsOfHistory() < s.MinMonthsHistory {
		return AlternativeScoringResult{
			Eligible: false,
			Reason:   fmt.Sprintf("insufficient history: need at least %d months, have %d", s.MinMonthsHistory, profile.TotalMonthsOfHistory()),
		}, nil
	}

	var factors []ScoringFactor

	// Factor 1: Payment consistency (40% weight)
	consistencyScore := s.scoreConsistency(profile)
	factors = append(factors, ScoringFactor{
		Name:   "payment_consistency",
		Weight: decimal.NewFromFloat(0.40),
		Score:  consistencyScore,
		Impact: impactLabel(consistencyScore),
	})

	// Factor 2: History length (25% weight)
	historyScore := s.scoreHistoryLength(profile)
	factors = append(factors, ScoringFactor{
		Name:   "history_length",
		Weight: decimal.NewFromFloat(0.25),
		Score:  historyScore,
		Impact: impactLabel(historyScore),
	})

	// Factor 3: Data source diversity (20% weight)
	diversityScore := s.scoreDiversity(profile)
	factors = append(factors, ScoringFactor{
		Name:   "data_diversity",
		Weight: decimal.NewFromFloat(0.20),
		Score:  diversityScore,
		Impact: impactLabel(diversityScore),
	})

	// Factor 4: Payment amount stability (15% weight)
	stabilityScore := s.scoreStability(profile)
	factors = append(factors, ScoringFactor{
		Name:   "payment_stability",
		Weight: decimal.NewFromFloat(0.15),
		Score:  stabilityScore,
		Impact: impactLabel(stabilityScore),
	})

	// Compute weighted total
	totalWeightedScore := decimal.Zero
	for _, f := range factors {
		totalWeightedScore = totalWeightedScore.Add(
			f.Weight.Mul(decimal.NewFromInt(int64(f.Score))),
		)
	}

	// Map to 300-850 range
	// Factor scores are 0-100, weighted sum is 0-100, map to 300-850
	rawScore := totalWeightedScore.IntPart()
	finalScore := 300 + int(rawScore*550/100)
	if finalScore > 850 {
		finalScore = 850
	}
	if finalScore < 300 {
		finalScore = 300
	}

	confidence := "MEDIUM"
	if profile.RecordCount() >= 4 && profile.TotalMonthsOfHistory() >= 24 {
		confidence = "HIGH"
	} else if profile.RecordCount() < 3 || profile.TotalMonthsOfHistory() < 12 {
		confidence = "LOW"
	}

	return AlternativeScoringResult{
		Score:      finalScore,
		Confidence: confidence,
		Factors:    factors,
		Eligible:   true,
		Reason:     fmt.Sprintf("alternative score computed from %d data sources", profile.RecordCount()),
	}, nil
}

// scoreConsistency evaluates payment on-time rates (0-100).
func (s *AlternativeScoring) scoreConsistency(profile valueobject.AlternativeCreditProfile) int {
	avgRate := profile.AverageOnTimeRate()
	return int(avgRate.IntPart())
}

// scoreHistoryLength evaluates the depth of payment history (0-100).
func (s *AlternativeScoring) scoreHistoryLength(profile valueobject.AlternativeCreditProfile) int {
	months := profile.TotalMonthsOfHistory()
	// 48+ months = 100, scale linearly
	if months >= 48 {
		return 100
	}
	return months * 100 / 48
}

// scoreDiversity evaluates the variety of data sources (0-100).
func (s *AlternativeScoring) scoreDiversity(profile valueobject.AlternativeCreditProfile) int {
	uniqueTypes := make(map[valueobject.AlternativeDataType]bool)
	for _, r := range profile.Records {
		uniqueTypes[r.DataType()] = true
	}
	count := len(uniqueTypes)
	// 4+ unique types = 100
	if count >= 4 {
		return 100
	}
	return count * 25
}

// scoreStability evaluates payment amount regularity (0-100).
// A simple heuristic: records with higher on-time rates suggest stability.
func (s *AlternativeScoring) scoreStability(profile valueobject.AlternativeCreditProfile) int {
	if profile.RecordCount() == 0 {
		return 0
	}

	totalRate := decimal.Zero
	for _, r := range profile.Records {
		totalRate = totalRate.Add(r.OnTimeRate())
	}
	avgRate := totalRate.Div(decimal.NewFromInt(int64(profile.RecordCount())))

	// Use on-time rate as a proxy for stability, scaled slightly
	score := int(avgRate.IntPart())
	if score > 100 {
		score = 100
	}
	return score
}

// Evaluate performs alternative scoring and returns a result compatible with
// the UnderwritingEngine. This integrates alternative scoring into the
// existing underwriting pipeline.
func (s *AlternativeScoring) Evaluate(
	profile valueobject.AlternativeCreditProfile,
	requestedAmount decimal.Decimal,
	termMonths int,
) UnderwritingResult {
	result, err := s.Score(profile)
	if err != nil {
		return UnderwritingResult{
			Approved: false,
			Reason:   fmt.Sprintf("alternative scoring failed: %v", err),
		}
	}

	if !result.Eligible {
		return UnderwritingResult{
			Approved: false,
			Reason:   result.Reason,
		}
	}

	// Use the standard underwriting engine with the alternative score.
	engine := NewUnderwritingEngine()
	creditScore := fmt.Sprintf("%d", result.Score)
	uwResult := engine.Evaluate(creditScore, requestedAmount, termMonths)

	// Adjust for lower confidence: reduce max amount for LOW confidence.
	if result.Confidence == "LOW" && uwResult.Approved {
		halfMax := uwResult.MaxAmount.Div(decimal.NewFromInt(2))
		if requestedAmount.GreaterThan(halfMax) {
			uwResult.Approved = false
			uwResult.Reason = "requested amount exceeds maximum for low-confidence alternative score"
		}
		uwResult.MaxAmount = halfMax
	}

	return uwResult
}

// impactLabel returns a human-readable impact label for a factor score.
func impactLabel(score int) string {
	switch {
	case score >= 80:
		return "POSITIVE"
	case score >= 50:
		return "NEUTRAL"
	default:
		return "NEGATIVE"
	}
}
