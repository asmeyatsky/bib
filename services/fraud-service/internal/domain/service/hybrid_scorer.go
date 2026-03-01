package service

import (
	"context"
	"log/slog"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/port"
)

// HybridScorer combines rule-based scoring with ML model predictions.
// If the ML model fails, it falls back to rules-only scoring.
type HybridScorer struct {
	rules    *RiskScorer
	ml       port.MLModelClient
	mlWeight float64
	logger   *slog.Logger
}

// NewHybridScorer creates a HybridScorer with the given ML weight (0.0–1.0).
// A weight of 0.0 means rules-only; 1.0 means ML-only.
func NewHybridScorer(rules *RiskScorer, ml port.MLModelClient, mlWeight float64, logger *slog.Logger) *HybridScorer {
	return &HybridScorer{
		rules:    rules,
		ml:       ml,
		mlWeight: mlWeight,
		logger:   logger,
	}
}

// Score evaluates risk using both rule-based and ML scoring, blending results.
func (h *HybridScorer) Score(input RiskInput) RiskOutput {
	// Always run rules first.
	rulesOutput := h.rules.Score(input)

	// Attempt ML prediction.
	features := map[string]interface{}{
		"amount":           input.Amount.InexactFloat64(),
		"currency":         input.Currency,
		"transaction_type": input.TransactionType,
		"account_id":       input.AccountID.String(),
	}
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			features["meta_"+k] = v
		}
	}

	mlScore, err := h.ml.Predict(context.Background(), features)
	if err != nil {
		h.logger.Warn("ML prediction failed, using rules-only scoring", "error", err)
		return rulesOutput
	}

	// Blend scores: combined = (1 - mlWeight) * rules + mlWeight * ml
	mlScoreInt := int(mlScore * 100)
	combined := int(float64(rulesOutput.Score)*(1-h.mlWeight) + float64(mlScoreInt)*h.mlWeight)

	// Cap at 100.
	if combined > 100 {
		combined = 100
	}

	signals := make([]string, len(rulesOutput.Signals))
	copy(signals, rulesOutput.Signals)
	signals = append(signals, "ml_enhanced")

	return RiskOutput{
		Score:   combined,
		Signals: signals,
	}
}
