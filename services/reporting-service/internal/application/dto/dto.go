package dto

import (
	"time"

	"github.com/google/uuid"
)

// GenerateReportRequest holds the input for generating a report.
type GenerateReportRequest struct {
	ReportType string    `json:"report_type"`
	Period     string    `json:"period"`
	TenantID   uuid.UUID `json:"tenant_id"`
}

// GenerateReportResponse holds the output after generating a report.
type GenerateReportResponse struct {
	ReportType      string    `json:"report_type"`
	ReportingPeriod string    `json:"reporting_period"`
	Status          string    `json:"status"`
	GeneratedAt     string    `json:"generated_at,omitempty"`
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
}

// GetReportRequest holds the input for retrieving a report.
type GetReportRequest struct {
	ID uuid.UUID `json:"id"`
}

// GetReportResponse holds the full report submission data.
type GetReportResponse struct {
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	GeneratedAt      *time.Time `json:"generated_at,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	ReportType       string     `json:"report_type"`
	ReportingPeriod  string     `json:"reporting_period"`
	Status           string     `json:"status"`
	XBRLContent      string     `json:"xbrl_content,omitempty"`
	ValidationErrors []string   `json:"validation_errors,omitempty"`
	Version          int        `json:"version"`
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
}

// SubmitReportRequest holds the input for submitting a report to the regulator.
type SubmitReportRequest struct {
	ID uuid.UUID `json:"id"`
}

// SubmitReportResponse holds the output after submitting a report.
type SubmitReportResponse struct {
	Status      string    `json:"status"`
	SubmittedAt string    `json:"submitted_at"`
	ID          uuid.UUID `json:"id"`
}
