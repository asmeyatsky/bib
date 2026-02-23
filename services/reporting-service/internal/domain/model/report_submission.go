package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

// ReportSubmission is the aggregate root for regulatory report submissions.
type ReportSubmission struct {
	id               uuid.UUID
	tenantID         uuid.UUID
	reportType       valueobject.ReportType
	reportingPeriod  string
	status           valueobject.SubmissionStatus
	xbrlContent      string
	generatedAt      *time.Time
	submittedAt      *time.Time
	validationErrors []string
	version          int
	createdAt        time.Time
	updatedAt        time.Time
	domainEvents     []events.DomainEvent
}

// NewReportSubmission creates a new ReportSubmission in DRAFT status.
func NewReportSubmission(tenantID uuid.UUID, reportType valueobject.ReportType, period string) (ReportSubmission, error) {
	if tenantID == uuid.Nil {
		return ReportSubmission{}, fmt.Errorf("tenant ID must not be nil")
	}
	if reportType.IsZero() {
		return ReportSubmission{}, fmt.Errorf("report type must not be empty")
	}
	if period == "" {
		return ReportSubmission{}, fmt.Errorf("reporting period must not be empty")
	}

	now := time.Now().UTC()
	return ReportSubmission{
		id:               uuid.New(),
		tenantID:         tenantID,
		reportType:       reportType,
		reportingPeriod:  period,
		status:           valueobject.SubmissionStatusDraft,
		xbrlContent:      "",
		validationErrors: []string{},
		version:          1,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// Reconstruct recreates a ReportSubmission from persisted data without emitting events.
func Reconstruct(
	id uuid.UUID,
	tenantID uuid.UUID,
	reportType valueobject.ReportType,
	reportingPeriod string,
	status valueobject.SubmissionStatus,
	xbrlContent string,
	generatedAt *time.Time,
	submittedAt *time.Time,
	validationErrors []string,
	version int,
	createdAt time.Time,
	updatedAt time.Time,
) ReportSubmission {
	if validationErrors == nil {
		validationErrors = []string{}
	}
	return ReportSubmission{
		id:               id,
		tenantID:         tenantID,
		reportType:       reportType,
		reportingPeriod:  reportingPeriod,
		status:           status,
		xbrlContent:      xbrlContent,
		generatedAt:      generatedAt,
		submittedAt:      submittedAt,
		validationErrors: validationErrors,
		version:          version,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

// MarkGenerating transitions from DRAFT to GENERATING.
func (r ReportSubmission) MarkGenerating(now time.Time) (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusDraft) {
		return r, fmt.Errorf("cannot mark generating: current status is %s, expected DRAFT", r.status)
	}
	r.status = valueobject.SubmissionStatusGenerating
	r.updatedAt = now
	return r, nil
}

// SetGenerated transitions from GENERATING to READY after XBRL content has been produced.
func (r ReportSubmission) SetGenerated(xbrlContent string, now time.Time) (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusGenerating) {
		return r, fmt.Errorf("cannot set generated: current status is %s, expected GENERATING", r.status)
	}
	if xbrlContent == "" {
		return r, fmt.Errorf("XBRL content must not be empty")
	}
	r.status = valueobject.SubmissionStatusReady
	r.xbrlContent = xbrlContent
	r.generatedAt = &now
	r.updatedAt = now
	r.domainEvents = append(r.domainEvents, event.NewReportGenerated(
		r.id, r.tenantID, r.reportType.String(), r.reportingPeriod, now,
	))
	return r, nil
}

// Validate performs basic XBRL validation on the content.
func (r ReportSubmission) Validate() (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusReady) {
		return r, fmt.Errorf("cannot validate: current status is %s, expected READY", r.status)
	}

	var errors []string

	if r.xbrlContent == "" {
		errors = append(errors, "XBRL content is empty")
	}
	if !strings.Contains(r.xbrlContent, "<?xml") {
		errors = append(errors, "XBRL content missing XML declaration")
	}
	if !strings.Contains(r.xbrlContent, "xbrli:xbrl") {
		errors = append(errors, "XBRL content missing xbrli:xbrl root element")
	}
	if !strings.Contains(r.xbrlContent, "xbrli:context") {
		errors = append(errors, "XBRL content missing xbrli:context element")
	}
	if !strings.Contains(r.xbrlContent, "xbrli:period") {
		errors = append(errors, "XBRL content missing xbrli:period element")
	}

	if len(errors) > 0 {
		r.validationErrors = errors
		return r, fmt.Errorf("XBRL validation failed: %s", strings.Join(errors, "; "))
	}

	r.validationErrors = []string{}
	return r, nil
}

// Submit transitions from READY to SUBMITTED.
func (r ReportSubmission) Submit(now time.Time) (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusReady) {
		return r, fmt.Errorf("cannot submit: current status is %s, expected READY", r.status)
	}
	r.status = valueobject.SubmissionStatusSubmitted
	r.submittedAt = &now
	r.updatedAt = now
	r.domainEvents = append(r.domainEvents, event.NewReportSubmitted(
		r.id, r.tenantID, r.reportType.String(), r.reportingPeriod, now,
	))
	return r, nil
}

// Accept transitions from SUBMITTED to ACCEPTED.
func (r ReportSubmission) Accept(now time.Time) (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusSubmitted) {
		return r, fmt.Errorf("cannot accept: current status is %s, expected SUBMITTED", r.status)
	}
	r.status = valueobject.SubmissionStatusAccepted
	r.updatedAt = now
	r.domainEvents = append(r.domainEvents, event.NewReportAccepted(
		r.id, r.tenantID, r.reportType.String(), r.reportingPeriod, now,
	))
	return r, nil
}

// Reject transitions from SUBMITTED to REJECTED with validation errors.
func (r ReportSubmission) Reject(errors []string, now time.Time) (ReportSubmission, error) {
	if !r.status.Equal(valueobject.SubmissionStatusSubmitted) {
		return r, fmt.Errorf("cannot reject: current status is %s, expected SUBMITTED", r.status)
	}
	if len(errors) == 0 {
		return r, fmt.Errorf("rejection must include at least one error")
	}
	r.status = valueobject.SubmissionStatusRejected
	r.validationErrors = errors
	r.updatedAt = now
	r.domainEvents = append(r.domainEvents, event.NewReportRejected(
		r.id, r.tenantID, r.reportType.String(), r.reportingPeriod, errors, now,
	))
	return r, nil
}

// --- Accessors ---

func (r ReportSubmission) ID() uuid.UUID                          { return r.id }
func (r ReportSubmission) TenantID() uuid.UUID                    { return r.tenantID }
func (r ReportSubmission) ReportType() valueobject.ReportType     { return r.reportType }
func (r ReportSubmission) ReportingPeriod() string                { return r.reportingPeriod }
func (r ReportSubmission) Status() valueobject.SubmissionStatus   { return r.status }
func (r ReportSubmission) XBRLContent() string                    { return r.xbrlContent }
func (r ReportSubmission) GeneratedAt() *time.Time                { return r.generatedAt }
func (r ReportSubmission) SubmittedAt() *time.Time                { return r.submittedAt }
func (r ReportSubmission) ValidationErrors() []string             { return r.validationErrors }
func (r ReportSubmission) Version() int                           { return r.version }
func (r ReportSubmission) CreatedAt() time.Time                   { return r.createdAt }
func (r ReportSubmission) UpdatedAt() time.Time                   { return r.updatedAt }

// DomainEvents returns the uncommitted domain events.
func (r ReportSubmission) DomainEvents() []events.DomainEvent {
	return r.domainEvents
}

// ClearDomainEvents returns a copy with cleared domain events.
func (r ReportSubmission) ClearDomainEvents() ReportSubmission {
	r.domainEvents = nil
	return r
}
