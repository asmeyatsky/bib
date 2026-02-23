package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/domain/port"
	"github.com/bibbank/bib/services/card-service/internal/domain/service"
)

// AuthorizeTransactionUseCase handles card transaction authorization with JIT funding.
type AuthorizeTransactionUseCase struct {
	cardRepo       port.CardRepository
	eventPublisher port.EventPublisher
	balanceClient  port.AccountBalanceClient
	jitFunding     *service.JITFundingService
}

// NewAuthorizeTransactionUseCase creates a new AuthorizeTransactionUseCase.
func NewAuthorizeTransactionUseCase(
	cardRepo port.CardRepository,
	eventPublisher port.EventPublisher,
	balanceClient port.AccountBalanceClient,
	jitFunding *service.JITFundingService,
) *AuthorizeTransactionUseCase {
	return &AuthorizeTransactionUseCase{
		cardRepo:       cardRepo,
		eventPublisher: eventPublisher,
		balanceClient:  balanceClient,
		jitFunding:     jitFunding,
	}
}

// Execute authorizes a card transaction.
// Flow: check JIT funding -> authorize on card aggregate -> persist -> publish events.
func (uc *AuthorizeTransactionUseCase) Execute(ctx context.Context, req dto.AuthorizeTransactionRequest) (dto.AuthorizeTransactionResponse, error) {
	// 1. Retrieve the card.
	card, err := uc.cardRepo.FindByID(ctx, req.CardID)
	if err != nil {
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   "card not found",
		}, fmt.Errorf("failed to find card: %w", err)
	}

	// 2. JIT Funding: check available balance on the linked account.
	availableBalance, err := uc.balanceClient.GetAvailableBalance(ctx, card.AccountID())
	if err != nil {
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   "unable to verify funds",
		}, fmt.Errorf("failed to get available balance: %w", err)
	}

	fundingResult := uc.jitFunding.CheckFunding(availableBalance, req.Amount)
	if !fundingResult.Approved {
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   fundingResult.DeclineReason,
		}, nil
	}

	// 3. Authorize on the card aggregate (checks status, expiry, limits).
	now := time.Now().UTC()
	updatedCard, authCode, err := card.AuthorizeTransaction(
		req.Amount,
		req.MerchantName,
		req.MerchantCategory,
		now,
	)
	if err != nil {
		// Publish decline events even on failure.
		_ = uc.eventPublisher.Publish(ctx, updatedCard.DomainEvents()) //nolint:errcheck
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   err.Error(),
		}, nil
	}

	// 4. Persist the updated card and transaction record.
	if err := uc.cardRepo.Update(ctx, updatedCard); err != nil {
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   "internal error",
		}, fmt.Errorf("failed to update card: %w", err)
	}

	if err := uc.cardRepo.SaveTransaction(
		ctx,
		updatedCard.ID(),
		req.Amount,
		req.Currency,
		req.MerchantName,
		req.MerchantCategory,
		authCode,
		"AUTHORIZED",
	); err != nil {
		return dto.AuthorizeTransactionResponse{
			Approved: false,
			Reason:   "internal error",
		}, fmt.Errorf("failed to save transaction: %w", err)
	}

	// 5. Publish domain events.
	if err := uc.eventPublisher.Publish(ctx, updatedCard.DomainEvents()); err != nil {
		// Log but don't fail the authorization -- transaction is committed.
		_ = err
	}

	return dto.AuthorizeTransactionResponse{
		Approved: true,
		AuthCode: authCode,
	}, nil
}
