package event

import (
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

// DomainEvent is an alias for the shared pkg/events.DomainEvent interface.
type DomainEvent = events.DomainEvent

const (
	// EventTypeAssessmentCompleted is emitted when a transaction assessment finishes.
	EventTypeAssessmentCompleted = "fraud.assessment.completed"

	// EventTypeHighRiskDetected is emitted when a CRITICAL risk level is detected.
	EventTypeHighRiskDetected = "fraud.high_risk.detected"
)

// AssessmentCompleted is published when a fraud assessment has been completed
// for a transaction.
type AssessmentCompleted struct {
	events.BaseEvent
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	RiskScore     int       `json:"risk_score"`
	RiskLevel     string    `json:"risk_level"`
	Decision      string    `json:"decision"`
	Signals       []string  `json:"signals"`
	AssessedAt    time.Time `json:"assessed_at"`
}

func NewAssessmentCompleted(assessmentID, tenantID, transactionID, accountID uuid.UUID, riskScore int, riskLevel, decision string, signals []string, assessedAt time.Time) AssessmentCompleted {
	return AssessmentCompleted{
		BaseEvent:     events.NewBaseEvent(EventTypeAssessmentCompleted, assessmentID.String(), "FraudAssessment", tenantID.String()),
		AssessmentID:  assessmentID,
		TransactionID: transactionID,
		AccountID:     accountID,
		RiskScore:     riskScore,
		RiskLevel:     riskLevel,
		Decision:      decision,
		Signals:       signals,
		AssessedAt:    assessedAt,
	}
}

// HighRiskDetected is published when a transaction is assessed with CRITICAL
// risk level, triggering alerts and potential account freezes.
type HighRiskDetected struct {
	events.BaseEvent
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	RiskScore     int       `json:"risk_score"`
	Signals       []string  `json:"signals"`
	DetectedAt    time.Time `json:"detected_at"`
}

func NewHighRiskDetected(assessmentID, tenantID, transactionID, accountID uuid.UUID, riskScore int, signals []string, detectedAt time.Time) HighRiskDetected {
	return HighRiskDetected{
		BaseEvent:     events.NewBaseEvent(EventTypeHighRiskDetected, assessmentID.String(), "FraudAssessment", tenantID.String()),
		AssessmentID:  assessmentID,
		TransactionID: transactionID,
		AccountID:     accountID,
		RiskScore:     riskScore,
		Signals:       signals,
		DetectedAt:    detectedAt,
	}
}
