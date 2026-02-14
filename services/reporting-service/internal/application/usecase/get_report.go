package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/port"
)

// GetReportUseCase retrieves a report submission by ID.
type GetReportUseCase struct {
	repo port.ReportSubmissionRepository
}

// NewGetReportUseCase creates a new GetReportUseCase.
func NewGetReportUseCase(repo port.ReportSubmissionRepository) *GetReportUseCase {
	return &GetReportUseCase{
		repo: repo,
	}
}

// Execute retrieves a report submission.
func (uc *GetReportUseCase) Execute(ctx context.Context, req dto.GetReportRequest) (dto.GetReportResponse, error) {
	submission, err := uc.repo.FindByID(ctx, req.ID)
	if err != nil {
		return dto.GetReportResponse{}, fmt.Errorf("failed to find report submission: %w", err)
	}

	return dto.GetReportResponse{
		ID:               submission.ID(),
		TenantID:         submission.TenantID(),
		ReportType:       submission.ReportType().String(),
		ReportingPeriod:  submission.ReportingPeriod(),
		Status:           submission.Status().String(),
		XBRLContent:      submission.XBRLContent(),
		GeneratedAt:      submission.GeneratedAt(),
		SubmittedAt:      submission.SubmittedAt(),
		ValidationErrors: submission.ValidationErrors(),
		Version:          submission.Version(),
		CreatedAt:        submission.CreatedAt(),
		UpdatedAt:        submission.UpdatedAt(),
	}, nil
}
