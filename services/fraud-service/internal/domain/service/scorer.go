package service

// Scorer defines the interface for risk scoring strategies.
// Both RiskScorer (rule-based) and HybridScorer (rules + ML) implement this.
type Scorer interface {
	Score(input RiskInput) RiskOutput
}
