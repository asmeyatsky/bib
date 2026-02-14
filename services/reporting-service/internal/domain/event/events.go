package event

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the interface that all domain events must implement.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() uuid.UUID
}

// ReportGenerated is emitted when a report's XBRL content has been generated.
type ReportGenerated struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	ReportType      string    `json:"report_type"`
	ReportingPeriod string    `json:"reporting_period"`
	Timestamp       time.Time `json:"timestamp"`
}

func (e ReportGenerated) EventType() string      { return "report.generated" }
func (e ReportGenerated) OccurredAt() time.Time   { return e.Timestamp }
func (e ReportGenerated) AggregateID() uuid.UUID  { return e.ID }

// ReportSubmitted is emitted when a report has been submitted to a regulatory authority.
type ReportSubmitted struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	ReportType      string    `json:"report_type"`
	ReportingPeriod string    `json:"reporting_period"`
	Timestamp       time.Time `json:"timestamp"`
}

func (e ReportSubmitted) EventType() string      { return "report.submitted" }
func (e ReportSubmitted) OccurredAt() time.Time   { return e.Timestamp }
func (e ReportSubmitted) AggregateID() uuid.UUID  { return e.ID }

// ReportAccepted is emitted when a submitted report has been accepted by the regulator.
type ReportAccepted struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	ReportType      string    `json:"report_type"`
	ReportingPeriod string    `json:"reporting_period"`
	Timestamp       time.Time `json:"timestamp"`
}

func (e ReportAccepted) EventType() string      { return "report.accepted" }
func (e ReportAccepted) OccurredAt() time.Time   { return e.Timestamp }
func (e ReportAccepted) AggregateID() uuid.UUID  { return e.ID }

// ReportRejected is emitted when a submitted report has been rejected by the regulator.
type ReportRejected struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	ReportType       string    `json:"report_type"`
	ReportingPeriod  string    `json:"reporting_period"`
	ValidationErrors []string  `json:"validation_errors"`
	Timestamp        time.Time `json:"timestamp"`
}

func (e ReportRejected) EventType() string      { return "report.rejected" }
func (e ReportRejected) OccurredAt() time.Time   { return e.Timestamp }
func (e ReportRejected) AggregateID() uuid.UUID  { return e.ID }
