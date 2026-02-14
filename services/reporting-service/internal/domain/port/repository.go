package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
)

// ReportSubmissionRepository defines the persistence port for report submissions.
type ReportSubmissionRepository interface {
	// Save persists a new or updated report submission.
	Save(ctx context.Context, submission model.ReportSubmission) error
	// FindByID retrieves a report submission by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (model.ReportSubmission, error)
	// FindByTenantAndPeriod retrieves report submissions for a tenant and period.
	FindByTenantAndPeriod(ctx context.Context, tenantID uuid.UUID, period string) ([]model.ReportSubmission, error)
	// FindByTenantAndType retrieves report submissions for a tenant and type.
	FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, reportType string) ([]model.ReportSubmission, error)
}

// EventPublisher defines the port for publishing domain events.
type EventPublisher interface {
	// Publish publishes one or more domain events.
	Publish(ctx context.Context, events ...event.DomainEvent) error
}

// LedgerDataClient defines the port for retrieving financial data from the ledger service.
type LedgerDataClient interface {
	// GetFinancialData retrieves aggregated financial data for a tenant and reporting period.
	GetFinancialData(ctx context.Context, tenantID uuid.UUID, period string) (service.ReportData, error)
}
