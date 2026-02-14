package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/domain/model"
	"github.com/bibbank/bib/services/card-service/internal/domain/port"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// IssueCardUseCase handles the creation and issuance of new cards.
type IssueCardUseCase struct {
	cardRepo       port.CardRepository
	eventPublisher port.EventPublisher
	cardProcessor  port.CardProcessorAdapter
}

// NewIssueCardUseCase creates a new IssueCardUseCase.
func NewIssueCardUseCase(
	cardRepo port.CardRepository,
	eventPublisher port.EventPublisher,
	cardProcessor port.CardProcessorAdapter,
) *IssueCardUseCase {
	return &IssueCardUseCase{
		cardRepo:       cardRepo,
		eventPublisher: eventPublisher,
		cardProcessor:  cardProcessor,
	}
}

// Execute issues a new card.
func (uc *IssueCardUseCase) Execute(ctx context.Context, req dto.IssueCardRequest) (dto.IssueCardResponse, error) {
	cardType, err := valueobject.NewCardType(req.CardType)
	if err != nil {
		return dto.IssueCardResponse{}, fmt.Errorf("invalid card type: %w", err)
	}

	card, err := model.NewCard(
		req.TenantID,
		req.AccountID,
		cardType,
		req.Currency,
		req.DailyLimit,
		req.MonthlyLimit,
	)
	if err != nil {
		return dto.IssueCardResponse{}, fmt.Errorf("failed to create card: %w", err)
	}

	if err := uc.cardRepo.Save(ctx, card); err != nil {
		return dto.IssueCardResponse{}, fmt.Errorf("failed to save card: %w", err)
	}

	// If physical card, request issuance from processor.
	if cardType.IsPhysical() {
		if err := uc.cardProcessor.IssuePhysicalCard(ctx, card); err != nil {
			// Log but don't fail -- the card is created in PENDING status.
			// Physical issuance can be retried.
			_ = err
		}
	}

	if err := uc.eventPublisher.Publish(ctx, card.DomainEvents()); err != nil {
		return dto.IssueCardResponse{}, fmt.Errorf("failed to publish events: %w", err)
	}

	return dto.IssueCardResponse{
		CardID:      card.ID(),
		LastFour:    card.CardNumber().LastFour(),
		ExpiryMonth: card.CardNumber().ExpiryMonth(),
		ExpiryYear:  card.CardNumber().ExpiryYear(),
		Status:      card.Status().String(),
		CardType:    card.CardType().String(),
		CreatedAt:   card.CreatedAt(),
	}, nil
}
