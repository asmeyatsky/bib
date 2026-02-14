package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
)

// GetVerification retrieves a single identity verification by ID.
type GetVerification struct {
	repo port.VerificationRepository
}

func NewGetVerification(repo port.VerificationRepository) *GetVerification {
	return &GetVerification{repo: repo}
}

func (uc *GetVerification) Execute(ctx context.Context, req dto.GetVerificationRequest) (dto.VerificationResponse, error) {
	verification, err := uc.repo.FindByID(ctx, req.ID)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to find verification: %w", err)
	}

	return toVerificationResponse(verification), nil
}
