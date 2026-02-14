package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/port"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

// GenerateReportUseCase orchestrates the generation of a regulatory report.
type GenerateReportUseCase struct {
	repo           port.ReportSubmissionRepository
	eventPublisher port.EventPublisher
	ledgerClient   port.LedgerDataClient
	xbrlGenerator  *service.XBRLGenerator
}

// NewGenerateReportUseCase creates a new GenerateReportUseCase.
func NewGenerateReportUseCase(
	repo port.ReportSubmissionRepository,
	eventPublisher port.EventPublisher,
	ledgerClient port.LedgerDataClient,
	xbrlGenerator *service.XBRLGenerator,
) *GenerateReportUseCase {
	return &GenerateReportUseCase{
		repo:           repo,
		eventPublisher: eventPublisher,
		ledgerClient:   ledgerClient,
		xbrlGenerator:  xbrlGenerator,
	}
}

// Execute generates a report for the given request.
func (uc *GenerateReportUseCase) Execute(ctx context.Context, req dto.GenerateReportRequest) (dto.GenerateReportResponse, error) {
	// Validate report type.
	reportType, err := valueobject.NewReportType(req.ReportType)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("invalid report type: %w", err)
	}

	// Create a new submission in DRAFT.
	submission, err := model.NewReportSubmission(req.TenantID, reportType, req.Period)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to create report submission: %w", err)
	}

	// Mark as generating.
	now := time.Now().UTC()
	submission, err = submission.MarkGenerating(now)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to mark generating: %w", err)
	}

	// Fetch financial data from ledger.
	data, err := uc.ledgerClient.GetFinancialData(ctx, req.TenantID, req.Period)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to fetch financial data: %w", err)
	}

	// Generate XBRL content.
	xbrlContent, err := uc.xbrlGenerator.Generate(reportType, data)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to generate XBRL: %w", err)
	}

	// Set generated content.
	now = time.Now().UTC()
	submission, err = submission.SetGenerated(xbrlContent, now)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to set generated content: %w", err)
	}

	// Validate the generated XBRL.
	submission, err = submission.Validate()
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("XBRL validation failed: %w", err)
	}

	// Persist submission.
	if err := uc.repo.Save(ctx, submission); err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("failed to save report submission: %w", err)
	}

	// Publish domain events.
	if events := submission.DomainEvents(); len(events) > 0 {
		if err := uc.eventPublisher.Publish(ctx, events...); err != nil {
			return dto.GenerateReportResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	generatedAt := ""
	if submission.GeneratedAt() != nil {
		generatedAt = submission.GeneratedAt().Format(time.RFC3339)
	}

	return dto.GenerateReportResponse{
		ID:              submission.ID(),
		TenantID:        submission.TenantID(),
		ReportType:      submission.ReportType().String(),
		ReportingPeriod: submission.ReportingPeriod(),
		Status:          submission.Status().String(),
		GeneratedAt:     generatedAt,
	}, nil
}
