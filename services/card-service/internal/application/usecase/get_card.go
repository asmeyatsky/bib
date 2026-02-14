package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/domain/port"
)

// GetCardUseCase handles retrieval of card details.
type GetCardUseCase struct {
	cardRepo port.CardRepository
}

// NewGetCardUseCase creates a new GetCardUseCase.
func NewGetCardUseCase(cardRepo port.CardRepository) *GetCardUseCase {
	return &GetCardUseCase{
		cardRepo: cardRepo,
	}
}

// Execute retrieves card details by ID.
func (uc *GetCardUseCase) Execute(ctx context.Context, req dto.GetCardRequest) (dto.CardResponse, error) {
	card, err := uc.cardRepo.FindByID(ctx, req.CardID)
	if err != nil {
		return dto.CardResponse{}, fmt.Errorf("failed to find card: %w", err)
	}

	return dto.CardResponse{
		ID:           card.ID(),
		TenantID:     card.TenantID(),
		AccountID:    card.AccountID(),
		CardType:     card.CardType().String(),
		Status:       card.Status().String(),
		LastFour:     card.CardNumber().LastFour(),
		ExpiryMonth:  card.CardNumber().ExpiryMonth(),
		ExpiryYear:   card.CardNumber().ExpiryYear(),
		Currency:     card.Currency(),
		DailyLimit:   card.DailyLimit(),
		MonthlyLimit: card.MonthlyLimit(),
		DailySpent:   card.DailySpent(),
		MonthlySpent: card.MonthlySpent(),
		CreatedAt:    card.CreatedAt(),
		UpdatedAt:    card.UpdatedAt(),
	}, nil
}
