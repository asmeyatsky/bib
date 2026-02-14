package event

import (
	"time"

	"github.com/google/uuid"
)

const (
	// EventTypeAssessmentCompleted is emitted when a transaction assessment finishes.
	EventTypeAssessmentCompleted = "fraud.assessment.completed"

	// EventTypeHighRiskDetected is emitted when a CRITICAL risk level is detected.
	EventTypeHighRiskDetected = "fraud.high_risk.detected"
)

// AssessmentCompleted is published when a fraud assessment has been completed
// for a transaction.
type AssessmentCompleted struct {
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	RiskScore     int       `json:"risk_score"`
	RiskLevel     string    `json:"risk_level"`
	Decision      string    `json:"decision"`
	Signals       []string  `json:"signals"`
	AssessedAt    time.Time `json:"assessed_at"`
}

// EventType returns the event type identifier.
func (e AssessmentCompleted) EventType() string {
	return EventTypeAssessmentCompleted
}

// AggregateID returns the assessment ID as the aggregate identifier.
func (e AssessmentCompleted) AggregateID() uuid.UUID {
	return e.AssessmentID
}

// HighRiskDetected is published when a transaction is assessed with CRITICAL
// risk level, triggering alerts and potential account freezes.
type HighRiskDetected struct {
	AssessmentID  uuid.UUID `json:"assessment_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	RiskScore     int       `json:"risk_score"`
	Signals       []string  `json:"signals"`
	DetectedAt    time.Time `json:"detected_at"`
}

// EventType returns the event type identifier.
func (e HighRiskDetected) EventType() string {
	return EventTypeHighRiskDetected
}

// AggregateID returns the assessment ID as the aggregate identifier.
func (e HighRiskDetected) AggregateID() uuid.UUID {
	return e.AssessmentID
}
