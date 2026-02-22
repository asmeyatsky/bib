package event

import (
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

// DomainEvent is an alias for the shared pkg/events.DomainEvent interface.
type DomainEvent = events.DomainEvent

// ReportGenerated is emitted when a report's XBRL content has been generated.
type ReportGenerated struct {
	events.BaseEvent
	ReportType      string `json:"report_type"`
	ReportingPeriod string `json:"reporting_period"`
}

func NewReportGenerated(id, tenantID uuid.UUID, reportType, reportingPeriod string, now time.Time) ReportGenerated {
	return ReportGenerated{
		BaseEvent:       events.NewBaseEvent("report.generated", id.String(), "ReportSubmission", tenantID.String()),
		ReportType:      reportType,
		ReportingPeriod: reportingPeriod,
	}
}

// ReportSubmitted is emitted when a report has been submitted to a regulatory authority.
type ReportSubmitted struct {
	events.BaseEvent
	ReportType      string `json:"report_type"`
	ReportingPeriod string `json:"reporting_period"`
}

func NewReportSubmitted(id, tenantID uuid.UUID, reportType, reportingPeriod string, now time.Time) ReportSubmitted {
	return ReportSubmitted{
		BaseEvent:       events.NewBaseEvent("report.submitted", id.String(), "ReportSubmission", tenantID.String()),
		ReportType:      reportType,
		ReportingPeriod: reportingPeriod,
	}
}

// ReportAccepted is emitted when a submitted report has been accepted by the regulator.
type ReportAccepted struct {
	events.BaseEvent
	ReportType      string `json:"report_type"`
	ReportingPeriod string `json:"reporting_period"`
}

func NewReportAccepted(id, tenantID uuid.UUID, reportType, reportingPeriod string, now time.Time) ReportAccepted {
	return ReportAccepted{
		BaseEvent:       events.NewBaseEvent("report.accepted", id.String(), "ReportSubmission", tenantID.String()),
		ReportType:      reportType,
		ReportingPeriod: reportingPeriod,
	}
}

// ReportRejected is emitted when a submitted report has been rejected by the regulator.
type ReportRejected struct {
	events.BaseEvent
	ReportType       string   `json:"report_type"`
	ReportingPeriod  string   `json:"reporting_period"`
	ValidationErrors []string `json:"validation_errors"`
}

func NewReportRejected(id, tenantID uuid.UUID, reportType, reportingPeriod string, validationErrors []string, now time.Time) ReportRejected {
	return ReportRejected{
		BaseEvent:        events.NewBaseEvent("report.rejected", id.String(), "ReportSubmission", tenantID.String()),
		ReportType:       reportType,
		ReportingPeriod:  reportingPeriod,
		ValidationErrors: validationErrors,
	}
}
