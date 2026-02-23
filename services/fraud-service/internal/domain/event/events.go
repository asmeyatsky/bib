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
	AssessedAt time.Time `json:"assessed_at"`
	events.BaseEvent
	RiskLevel     string    `json:"risk_level"`
	Decision      string    `json:"decision"`
	Signals       []string  `json:"signals"`
	RiskScore     int       `json:"risk_score"`
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
}

func NewAssessmentCompleted(assessmentID, tenantID, transactionID, accountID uuid.UUID, riskScore int, riskLevel, decision string, signals []string, assessedAt time.Time) AssessmentCompleted {
	return AssessmentCompleted{
		BaseEvent:     events.NewBaseEvent(EventTypeAssessmentCompleted, assessmentID.String(), "FraudAssessment", tenantID.String()),
		AssessedAt:    assessedAt,
		Signals:       signals,
		AssessmentID:  assessmentID,
		TransactionID: transactionID,
		AccountID:     accountID,
		RiskLevel:     riskLevel,
		Decision:      decision,
		RiskScore:     riskScore,
	}
}

// HighRiskDetected is published when a transaction is assessed with CRITICAL
// risk level, triggering alerts and potential account freezes.
type HighRiskDetected struct {
	events.BaseEvent
	DetectedAt    time.Time `json:"detected_at"`
	Signals       []string  `json:"signals"`
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	RiskScore     int       `json:"risk_score"`
}

func NewHighRiskDetected(assessmentID, tenantID, transactionID, accountID uuid.UUID, riskScore int, signals []string, detectedAt time.Time) HighRiskDetected {
	return HighRiskDetected{
		BaseEvent:     events.NewBaseEvent(EventTypeHighRiskDetected, assessmentID.String(), "FraudAssessment", tenantID.String()),
		DetectedAt:    detectedAt,
		Signals:       signals,
		AssessmentID:  assessmentID,
		TransactionID: transactionID,
		AccountID:     accountID,
		RiskScore:     riskScore,
	}
}
