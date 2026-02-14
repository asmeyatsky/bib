package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/domain/port"
)

// FreezeCardUseCase handles freezing a card.
type FreezeCardUseCase struct {
	cardRepo       port.CardRepository
	eventPublisher port.EventPublisher
}

// NewFreezeCardUseCase creates a new FreezeCardUseCase.
func NewFreezeCardUseCase(
	cardRepo port.CardRepository,
	eventPublisher port.EventPublisher,
) *FreezeCardUseCase {
	return &FreezeCardUseCase{
		cardRepo:       cardRepo,
		eventPublisher: eventPublisher,
	}
}

// Execute freezes a card.
func (uc *FreezeCardUseCase) Execute(ctx context.Context, req dto.FreezeCardRequest) (dto.FreezeCardResponse, error) {
	card, err := uc.cardRepo.FindByID(ctx, req.CardID)
	if err != nil {
		return dto.FreezeCardResponse{}, fmt.Errorf("failed to find card: %w", err)
	}

	now := time.Now().UTC()
	frozenCard, err := card.Freeze(now)
	if err != nil {
		return dto.FreezeCardResponse{}, fmt.Errorf("failed to freeze card: %w", err)
	}

	if err := uc.cardRepo.Update(ctx, frozenCard); err != nil {
		return dto.FreezeCardResponse{}, fmt.Errorf("failed to update card: %w", err)
	}

	if err := uc.eventPublisher.Publish(ctx, frozenCard.DomainEvents()); err != nil {
		// Log but do not fail.
		_ = err
	}

	return dto.FreezeCardResponse{
		CardID: frozenCard.ID(),
		Status: frozenCard.Status().String(),
	}, nil
}
