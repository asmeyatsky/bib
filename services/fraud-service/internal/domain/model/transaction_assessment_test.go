package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/event"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/valueobject"
)

func newValidAssessment(t *testing.T) *model.TransactionAssessment {
	t.Helper()
	a, err := model.NewTransactionAssessment(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		decimal.NewFromInt(5000),
		"USD",
		"transfer",
	)
	require.NoError(t, err)
	return a
}

func TestNewTransactionAssessment_Valid(t *testing.T) {
	a := newValidAssessment(t)

	assert.NotEqual(t, uuid.Nil, a.ID())
	assert.Equal(t, "USD", a.Currency())
	assert.Equal(t, "transfer", a.TransactionType())
	assert.Equal(t, 0, a.RiskScore())
	assert.Equal(t, 1, a.Version())
	assert.False(t, a.CreatedAt().IsZero())
}

func TestNewTransactionAssessment_Validation(t *testing.T) {
	tests := []struct {
		name      string
		amount    decimal.Decimal
		currency  string
		txnType   string
		wantErr   string
		tenantID  uuid.UUID
		txnID     uuid.UUID
		accountID uuid.UUID
	}{
		{
			name:      "nil tenant ID",
			txnID:     uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.NewFromInt(100),
			currency:  "USD",
			txnType:   "transfer",
			wantErr:   "tenant ID is required",
		},
		{
			name:      "nil transaction ID",
			tenantID:  uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.NewFromInt(100),
			currency:  "USD",
			txnType:   "transfer",
			wantErr:   "transaction ID is required",
		},
		{
			name:     "nil account ID",
			tenantID: uuid.New(),
			txnID:    uuid.New(),
			amount:   decimal.NewFromInt(100),
			currency: "USD",
			txnType:  "transfer",
			wantErr:  "account ID is required",
		},
		{
			name:      "zero amount",
			tenantID:  uuid.New(),
			txnID:     uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.Zero,
			currency:  "USD",
			txnType:   "transfer",
			wantErr:   "amount must be positive",
		},
		{
			name:      "negative amount",
			tenantID:  uuid.New(),
			txnID:     uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.NewFromInt(-100),
			currency:  "USD",
			txnType:   "transfer",
			wantErr:   "amount must be positive",
		},
		{
			name:      "empty currency",
			tenantID:  uuid.New(),
			txnID:     uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.NewFromInt(100),
			currency:  "",
			txnType:   "transfer",
			wantErr:   "currency is required",
		},
		{
			name:      "empty transaction type",
			tenantID:  uuid.New(),
			txnID:     uuid.New(),
			accountID: uuid.New(),
			amount:    decimal.NewFromInt(100),
			currency:  "USD",
			txnType:   "",
			wantErr:   "transaction type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.NewTransactionAssessment(
				tt.tenantID, tt.txnID, tt.accountID,
				tt.amount, tt.currency, tt.txnType,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAssess_LowRisk_Approve(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(15, []string{"normal_transaction"})
	require.NoError(t, err)

	assert.Equal(t, 15, a.RiskScore())
	assert.True(t, valueobject.RiskLevelLow.Equal(a.RiskLevel()))
	assert.True(t, valueobject.DecisionApprove.Equal(a.Decision()))
	assert.Equal(t, []string{"normal_transaction"}, a.RiskSignals())
	assert.False(t, a.AssessedAt().IsZero())
	assert.Equal(t, 2, a.Version())
}

func TestAssess_MediumRisk_Review(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(50, []string{"high_value", "cross_border"})
	require.NoError(t, err)

	assert.Equal(t, 50, a.RiskScore())
	assert.True(t, valueobject.RiskLevelMedium.Equal(a.RiskLevel()))
	assert.True(t, valueobject.DecisionReview.Equal(a.Decision()))
}

func TestAssess_HighRisk_Decline(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(75, []string{"high_value", "cross_border", "high_risk_country"})
	require.NoError(t, err)

	assert.Equal(t, 75, a.RiskScore())
	assert.True(t, valueobject.RiskLevelHigh.Equal(a.RiskLevel()))
	assert.True(t, valueobject.DecisionDecline.Equal(a.Decision()))
}

func TestAssess_CriticalRisk_Decline_EmitsHighRiskEvent(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(90, []string{"high_value", "crypto_transaction", "high_risk_country"})
	require.NoError(t, err)

	assert.Equal(t, 90, a.RiskScore())
	assert.True(t, valueobject.RiskLevelCritical.Equal(a.RiskLevel()))
	assert.True(t, valueobject.DecisionDecline.Equal(a.Decision()))

	// Should emit both AssessmentCompleted and HighRiskDetected events.
	events := a.DomainEvents()
	require.Len(t, events, 2)

	_, isAssessmentCompleted := events[0].(event.AssessmentCompleted)
	assert.True(t, isAssessmentCompleted)

	highRiskEvt, isHighRisk := events[1].(event.HighRiskDetected)
	assert.True(t, isHighRisk)
	assert.Equal(t, 90, highRiskEvt.RiskScore)
}

func TestAssess_BoundaryScores(t *testing.T) {
	tests := []struct {
		name     string
		decision valueobject.AssessmentDecision
		score    int
	}{
		{name: "score 0 approves", decision: valueobject.DecisionApprove, score: 0},
		{name: "score 29 approves", decision: valueobject.DecisionApprove, score: 29},
		{name: "score 30 reviews", decision: valueobject.DecisionReview, score: 30},
		{name: "score 70 reviews", decision: valueobject.DecisionReview, score: 70},
		{name: "score 71 declines", decision: valueobject.DecisionDecline, score: 71},
		{name: "score 100 declines", decision: valueobject.DecisionDecline, score: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newValidAssessment(t)
			err := a.Assess(tt.score, nil)
			require.NoError(t, err)
			assert.True(t, tt.decision.Equal(a.Decision()),
				"expected %s for score %d, got %s", tt.decision.String(), tt.score, a.Decision().String())
		})
	}
}

func TestAssess_InvalidScore(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(-1, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "risk score must be between 0 and 100")

	err = a.Assess(101, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "risk score must be between 0 and 100")
}

func TestAssess_EmitsAssessmentCompletedEvent(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(20, []string{"normal"})
	require.NoError(t, err)

	events := a.DomainEvents()
	require.Len(t, events, 1)

	evt, ok := events[0].(event.AssessmentCompleted)
	require.True(t, ok)
	assert.Equal(t, a.ID(), evt.AssessmentID)
	assert.Equal(t, a.TenantID().String(), evt.TenantID())
	assert.Equal(t, 20, evt.RiskScore)
	assert.Equal(t, "LOW", evt.RiskLevel)
	assert.Equal(t, "APPROVE", evt.Decision)
}

func TestDomainEvents_ClearsAfterRead(t *testing.T) {
	a := newValidAssessment(t)

	err := a.Assess(20, nil)
	require.NoError(t, err)

	events1 := a.DomainEvents()
	assert.Len(t, events1, 1)

	events2 := a.DomainEvents()
	assert.Len(t, events2, 0)
}
