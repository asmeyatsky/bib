package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/port"
)

// SubmitReportUseCase orchestrates the submission of a generated report to the regulator.
type SubmitReportUseCase struct {
	repo           port.ReportSubmissionRepository
	eventPublisher port.EventPublisher
}

// NewSubmitReportUseCase creates a new SubmitReportUseCase.
func NewSubmitReportUseCase(
	repo port.ReportSubmissionRepository,
	eventPublisher port.EventPublisher,
) *SubmitReportUseCase {
	return &SubmitReportUseCase{
		repo:           repo,
		eventPublisher: eventPublisher,
	}
}

// Execute submits a report to the regulatory authority.
func (uc *SubmitReportUseCase) Execute(ctx context.Context, req dto.SubmitReportRequest) (dto.SubmitReportResponse, error) {
	// Retrieve the submission.
	submission, err := uc.repo.FindByID(ctx, req.ID)
	if err != nil {
		return dto.SubmitReportResponse{}, fmt.Errorf("failed to find report submission: %w", err)
	}

	// Submit.
	now := time.Now().UTC()
	submission, err = submission.Submit(now)
	if err != nil {
		return dto.SubmitReportResponse{}, fmt.Errorf("failed to submit report: %w", err)
	}

	// Persist.
	if err := uc.repo.Save(ctx, submission); err != nil {
		return dto.SubmitReportResponse{}, fmt.Errorf("failed to save submitted report: %w", err)
	}

	// Publish domain events.
	if events := submission.DomainEvents(); len(events) > 0 {
		if err := uc.eventPublisher.Publish(ctx, events...); err != nil {
			return dto.SubmitReportResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	submittedAt := ""
	if submission.SubmittedAt() != nil {
		submittedAt = submission.SubmittedAt().Format(time.RFC3339)
	}

	return dto.SubmitReportResponse{
		ID:          submission.ID(),
		Status:      submission.Status().String(),
		SubmittedAt: submittedAt,
	}, nil
}
