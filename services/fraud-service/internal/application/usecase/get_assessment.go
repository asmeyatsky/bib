package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/port"
)

// GetAssessment is the use case for retrieving an existing assessment.
type GetAssessment struct {
	repo port.AssessmentRepository
}

// NewGetAssessment creates a new GetAssessment use case.
func NewGetAssessment(repo port.AssessmentRepository) *GetAssessment {
	return &GetAssessment{repo: repo}
}

// Execute retrieves a transaction assessment by ID.
func (uc *GetAssessment) Execute(ctx context.Context, req dto.GetAssessmentRequest) (dto.AssessmentResponse, error) {
	assessment, err := uc.repo.FindByID(ctx, req.TenantID, req.AssessmentID)
	if err != nil {
		return dto.AssessmentResponse{}, fmt.Errorf("failed to find assessment: %w", err)
	}
	if assessment == nil {
		return dto.AssessmentResponse{}, fmt.Errorf("assessment not found: %s", req.AssessmentID)
	}

	return dto.FromModel(assessment), nil
}
