package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
)

// GetDepositPosition handles fetching a single deposit position by ID.
type GetDepositPosition struct {
	positionRepo port.DepositPositionRepository
}

func NewGetDepositPosition(positionRepo port.DepositPositionRepository) *GetDepositPosition {
	return &GetDepositPosition{positionRepo: positionRepo}
}

func (uc *GetDepositPosition) Execute(ctx context.Context, req dto.GetPositionRequest) (dto.DepositPositionResponse, error) {
	position, err := uc.positionRepo.FindByID(ctx, req.PositionID)
	if err != nil {
		return dto.DepositPositionResponse{}, fmt.Errorf("failed to find position: %w", err)
	}

	return toPositionResponse(position), nil
}
