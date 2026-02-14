package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
)

// ListVerifications retrieves verifications for a tenant with pagination.
type ListVerifications struct {
	repo port.VerificationRepository
}

func NewListVerifications(repo port.VerificationRepository) *ListVerifications {
	return &ListVerifications{repo: repo}
}

func (uc *ListVerifications) Execute(ctx context.Context, req dto.ListVerificationsRequest) (dto.ListVerificationsResponse, error) {
	verifications, total, err := uc.repo.ListByTenant(ctx, req.TenantID, req.PageSize, req.Offset)
	if err != nil {
		return dto.ListVerificationsResponse{}, fmt.Errorf("failed to list verifications: %w", err)
	}

	var responses []dto.VerificationResponse
	for _, v := range verifications {
		responses = append(responses, toVerificationResponse(v))
	}

	return dto.ListVerificationsResponse{
		Verifications: responses,
		TotalCount:    total,
	}, nil
}
