package service_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
)

type mockMLClient struct {
	err   error
	score float64
}

func (m *mockMLClient) Predict(_ context.Context, _ map[string]interface{}) (float64, error) {
	return m.score, m.err
}

func TestHybridScorer_CombinedScoring(t *testing.T) {
	rules := service.NewRiskScorer()
	ml := &mockMLClient{score: 0.8} // ML says 80% risk
	logger := slog.Default()

	scorer := service.NewHybridScorer(rules, ml, 0.5, logger)

	input := service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		TransactionType: "transfer",
		AccountID:       uuid.New(),
	}

	output := scorer.Score(input)

	// Rules base = 10, ML = 80, weight 0.5 → (10*0.5 + 80*0.5) = 45
	assert.Equal(t, 45, output.Score)
	assert.Contains(t, output.Signals, "ml_enhanced")
}

func TestHybridScorer_FallbackOnMLError(t *testing.T) {
	rules := service.NewRiskScorer()
	ml := &mockMLClient{err: fmt.Errorf("model unavailable")}
	logger := slog.Default()

	scorer := service.NewHybridScorer(rules, ml, 0.5, logger)

	input := service.RiskInput{
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		TransactionType: "transfer",
		AccountID:       uuid.New(),
	}

	output := scorer.Score(input)

	// Should fall back to rules-only: base score 10
	assert.Equal(t, 10, output.Score)
	assert.NotContains(t, output.Signals, "ml_enhanced")
}

func TestHybridScorer_ZeroWeightEqualsRulesOnly(t *testing.T) {
	rules := service.NewRiskScorer()
	ml := &mockMLClient{score: 0.9}
	logger := slog.Default()

	scorer := service.NewHybridScorer(rules, ml, 0.0, logger)

	input := service.RiskInput{
		Amount:          decimal.NewFromInt(55000),
		Currency:        "USD",
		TransactionType: "wire_transfer",
		AccountID:       uuid.New(),
	}

	rulesOnly := rules.Score(input)
	hybrid := scorer.Score(input)

	// With weight 0.0, hybrid score should equal rules score
	require.Equal(t, rulesOnly.Score, hybrid.Score)
	// But still has ml_enhanced signal since ML was called successfully
	assert.Contains(t, hybrid.Signals, "ml_enhanced")
}
