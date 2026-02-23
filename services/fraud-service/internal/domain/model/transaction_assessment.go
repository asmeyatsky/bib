package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/event"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/valueobject"
)

// TransactionAssessment is the aggregate root for fraud risk assessments.
type TransactionAssessment struct {
	assessedAt      time.Time
	createdAt       time.Time
	updatedAt       time.Time
	currency        string
	amount          decimal.Decimal
	decision        valueobject.AssessmentDecision
	riskLevel       valueobject.RiskLevel
	transactionType string
	riskSignals     []string
	domainEvents    []events.DomainEvent
	riskScore       int
	version         int
	accountID       uuid.UUID
	transactionID   uuid.UUID
	tenantID        uuid.UUID
	id              uuid.UUID
}

// NewTransactionAssessment creates a new assessment for an incoming transaction.
// The assessment starts unscored; call Assess() to run scoring.
func NewTransactionAssessment(
	tenantID uuid.UUID,
	transactionID uuid.UUID,
	accountID uuid.UUID,
	amount decimal.Decimal,
	currency string,
	transactionType string,
) (*TransactionAssessment, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if transactionID == uuid.Nil {
		return nil, fmt.Errorf("transaction ID is required")
	}
	if accountID == uuid.Nil {
		return nil, fmt.Errorf("account ID is required")
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, fmt.Errorf("amount must be positive")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if transactionType == "" {
		return nil, fmt.Errorf("transaction type is required")
	}

	now := time.Now().UTC()

	return &TransactionAssessment{
		id:              uuid.New(),
		tenantID:        tenantID,
		transactionID:   transactionID,
		accountID:       accountID,
		amount:          amount,
		currency:        currency,
		transactionType: transactionType,
		riskLevel:       valueobject.RiskLevelLow,
		riskScore:       0,
		decision:        valueobject.AssessmentDecision{},
		riskSignals:     make([]string, 0),
		version:         1,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// Assess applies a risk score and signals to the assessment, determining the
// risk level and decision. This is the core domain operation.
func (a *TransactionAssessment) Assess(riskScore int, signals []string) error {
	if riskScore < 0 || riskScore > 100 {
		return fmt.Errorf("risk score must be between 0 and 100, got %d", riskScore)
	}

	a.riskScore = riskScore
	a.riskSignals = signals
	a.riskLevel = valueobject.RiskLevelFromScore(riskScore)
	a.decision = valueobject.DecisionFromScore(riskScore)
	a.assessedAt = time.Now().UTC()
	a.updatedAt = a.assessedAt
	a.version++

	// Emit AssessmentCompleted event.
	a.domainEvents = append(a.domainEvents, event.NewAssessmentCompleted(
		a.id, a.tenantID, a.transactionID, a.accountID,
		a.riskScore, a.riskLevel.String(), a.decision.String(),
		a.riskSignals, a.assessedAt,
	))

	// Emit HighRiskDetected if the risk level is CRITICAL.
	if a.riskLevel.Equal(valueobject.RiskLevelCritical) {
		a.domainEvents = append(a.domainEvents, event.NewHighRiskDetected(
			a.id, a.tenantID, a.transactionID, a.accountID,
			a.riskScore, a.riskSignals, a.assessedAt,
		))
	}

	return nil
}

// Reconstruct rebuilds a TransactionAssessment from persisted data (no validation, no events).
func Reconstruct(
	id, tenantID, transactionID, accountID uuid.UUID,
	amount decimal.Decimal,
	currency, transactionType string,
	riskLevel valueobject.RiskLevel,
	riskScore int,
	decision valueobject.AssessmentDecision,
	riskSignals []string,
	assessedAt time.Time,
	version int,
	createdAt, updatedAt time.Time,
) *TransactionAssessment {
	return &TransactionAssessment{
		id:              id,
		tenantID:        tenantID,
		transactionID:   transactionID,
		accountID:       accountID,
		amount:          amount,
		currency:        currency,
		transactionType: transactionType,
		riskLevel:       riskLevel,
		riskScore:       riskScore,
		decision:        decision,
		riskSignals:     riskSignals,
		assessedAt:      assessedAt,
		version:         version,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		domainEvents:    make([]events.DomainEvent, 0),
	}
}

// --- Accessors ---

func (a *TransactionAssessment) ID() uuid.UUID                            { return a.id }
func (a *TransactionAssessment) TenantID() uuid.UUID                      { return a.tenantID }
func (a *TransactionAssessment) TransactionID() uuid.UUID                 { return a.transactionID }
func (a *TransactionAssessment) AccountID() uuid.UUID                     { return a.accountID }
func (a *TransactionAssessment) Amount() decimal.Decimal                  { return a.amount }
func (a *TransactionAssessment) Currency() string                         { return a.currency }
func (a *TransactionAssessment) TransactionType() string                  { return a.transactionType }
func (a *TransactionAssessment) RiskLevel() valueobject.RiskLevel         { return a.riskLevel }
func (a *TransactionAssessment) RiskScore() int                           { return a.riskScore }
func (a *TransactionAssessment) Decision() valueobject.AssessmentDecision { return a.decision }
func (a *TransactionAssessment) RiskSignals() []string                    { return a.riskSignals }
func (a *TransactionAssessment) AssessedAt() time.Time                    { return a.assessedAt }
func (a *TransactionAssessment) Version() int                             { return a.version }
func (a *TransactionAssessment) CreatedAt() time.Time                     { return a.createdAt }
func (a *TransactionAssessment) UpdatedAt() time.Time                     { return a.updatedAt }

// DomainEvents returns all accumulated domain events and clears them.
func (a *TransactionAssessment) DomainEvents() []events.DomainEvent {
	evts := a.domainEvents
	a.domainEvents = make([]events.DomainEvent, 0)
	return evts
}
