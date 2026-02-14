package tests

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/bibbank/bib/services/lending-service/internal/domain/service"
)

func TestUnderwritingEngine_ExcellentCredit(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("780", decimal.NewFromInt(200_000), 360)

	assert.True(t, result.Approved)
	assert.Equal(t, "excellent credit tier", result.Reason)
	assert.Equal(t, "780", result.CreditScore)
	assert.Equal(t, 450, result.SuggestedRate)
	assert.True(t, result.MaxAmount.Equal(decimal.NewFromInt(500_000)))
}

func TestUnderwritingEngine_GoodCredit(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("720", decimal.NewFromInt(100_000), 240)

	assert.True(t, result.Approved)
	assert.Equal(t, "good credit tier", result.Reason)
	assert.Equal(t, 550, result.SuggestedRate)
	assert.True(t, result.MaxAmount.Equal(decimal.NewFromInt(250_000)))
}

func TestUnderwritingEngine_FairCredit(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("650", decimal.NewFromInt(50_000), 60)

	assert.True(t, result.Approved)
	assert.Contains(t, result.Reason, "fair credit tier")
	assert.Equal(t, 850, result.SuggestedRate)
	assert.True(t, result.MaxAmount.Equal(decimal.NewFromInt(100_000)))
}

func TestUnderwritingEngine_PoorCredit_Rejected(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("550", decimal.NewFromInt(10_000), 12)

	assert.False(t, result.Approved)
	assert.Equal(t, "credit score below minimum threshold", result.Reason)
	assert.Equal(t, 0, result.SuggestedRate)
}

func TestUnderwritingEngine_ExceedsMaxAmount(t *testing.T) {
	engine := service.NewUnderwritingEngine()

	// Score 720 => max $250K, requesting $300K.
	result := engine.Evaluate("720", decimal.NewFromInt(300_000), 360)

	assert.False(t, result.Approved)
	assert.Equal(t, "requested amount exceeds maximum for credit tier", result.Reason)
}

func TestUnderwritingEngine_ExceedsMaxTerm(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("780", decimal.NewFromInt(100_000), 480) // 40 years

	assert.False(t, result.Approved)
	assert.Equal(t, "term exceeds maximum 360 months", result.Reason)
}

func TestUnderwritingEngine_InvalidScore(t *testing.T) {
	engine := service.NewUnderwritingEngine()
	result := engine.Evaluate("not-a-number", decimal.NewFromInt(10_000), 12)

	assert.False(t, result.Approved)
	assert.Equal(t, "unable to parse credit score", result.Reason)
}

func TestUnderwritingEngine_BoundaryScores(t *testing.T) {
	engine := service.NewUnderwritingEngine()

	t.Run("score 750 is excellent", func(t *testing.T) {
		r := engine.Evaluate("750", decimal.NewFromInt(10_000), 12)
		assert.True(t, r.Approved)
		assert.Equal(t, 450, r.SuggestedRate)
	})

	t.Run("score 700 is good", func(t *testing.T) {
		r := engine.Evaluate("700", decimal.NewFromInt(10_000), 12)
		assert.True(t, r.Approved)
		assert.Equal(t, 550, r.SuggestedRate)
	})

	t.Run("score 600 is fair", func(t *testing.T) {
		r := engine.Evaluate("600", decimal.NewFromInt(10_000), 12)
		assert.True(t, r.Approved)
		assert.Equal(t, 850, r.SuggestedRate)
	})

	t.Run("score 599 is rejected", func(t *testing.T) {
		r := engine.Evaluate("599", decimal.NewFromInt(10_000), 12)
		assert.False(t, r.Approved)
	})
}
