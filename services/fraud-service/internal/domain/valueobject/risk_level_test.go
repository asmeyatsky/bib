package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/valueobject"
)

func TestRiskLevel_Score(t *testing.T) {
	tests := []struct {
		name     string
		level    valueobject.RiskLevel
		expected int
	}{
		{"LOW score is 25", valueobject.RiskLevelLow, 25},
		{"MEDIUM score is 50", valueobject.RiskLevelMedium, 50},
		{"HIGH score is 75", valueobject.RiskLevelHigh, 75},
		{"CRITICAL score is 100", valueobject.RiskLevelCritical, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.Score())
		})
	}
}

func TestRiskLevel_String(t *testing.T) {
	assert.Equal(t, "LOW", valueobject.RiskLevelLow.String())
	assert.Equal(t, "MEDIUM", valueobject.RiskLevelMedium.String())
	assert.Equal(t, "HIGH", valueobject.RiskLevelHigh.String())
	assert.Equal(t, "CRITICAL", valueobject.RiskLevelCritical.String())
}

func TestRiskLevel_FromString(t *testing.T) {
	tests := []struct {
		input    string
		expected valueobject.RiskLevel
		wantErr  bool
	}{
		{"LOW", valueobject.RiskLevelLow, false},
		{"MEDIUM", valueobject.RiskLevelMedium, false},
		{"HIGH", valueobject.RiskLevelHigh, false},
		{"CRITICAL", valueobject.RiskLevelCritical, false},
		{"INVALID", valueobject.RiskLevel{}, true},
		{"", valueobject.RiskLevel{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := valueobject.RiskLevelFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result))
			}
		})
	}
}

func TestRiskLevel_FromScore(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected valueobject.RiskLevel
	}{
		{"score 0 is LOW", 0, valueobject.RiskLevelLow},
		{"score 10 is LOW", 10, valueobject.RiskLevelLow},
		{"score 34 is LOW", 34, valueobject.RiskLevelLow},
		{"score 35 is MEDIUM", 35, valueobject.RiskLevelMedium},
		{"score 50 is MEDIUM", 50, valueobject.RiskLevelMedium},
		{"score 59 is MEDIUM", 59, valueobject.RiskLevelMedium},
		{"score 60 is HIGH", 60, valueobject.RiskLevelHigh},
		{"score 75 is HIGH", 75, valueobject.RiskLevelHigh},
		{"score 79 is HIGH", 79, valueobject.RiskLevelHigh},
		{"score 80 is CRITICAL", 80, valueobject.RiskLevelCritical},
		{"score 100 is CRITICAL", 100, valueobject.RiskLevelCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueobject.RiskLevelFromScore(tt.score)
			assert.True(t, tt.expected.Equal(result),
				"expected %s for score %d, got %s", tt.expected.String(), tt.score, result.String())
		})
	}
}

func TestRiskLevel_Equal(t *testing.T) {
	assert.True(t, valueobject.RiskLevelLow.Equal(valueobject.RiskLevelLow))
	assert.False(t, valueobject.RiskLevelLow.Equal(valueobject.RiskLevelHigh))
}

func TestRiskLevel_IsZero(t *testing.T) {
	var zero valueobject.RiskLevel
	assert.True(t, zero.IsZero())
	assert.False(t, valueobject.RiskLevelLow.IsZero())
}
