package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
)

// CardServiceHandler handles gRPC requests for the card service.
// This implements the gRPC server interface for card operations.
type CardServiceHandler struct {
	issueCardUC       *usecase.IssueCardUseCase
	authorizeUC       *usecase.AuthorizeTransactionUseCase
	getCardUC         *usecase.GetCardUseCase
	freezeCardUC      *usecase.FreezeCardUseCase
}

// NewCardServiceHandler creates a new CardServiceHandler.
func NewCardServiceHandler(
	issueCardUC *usecase.IssueCardUseCase,
	authorizeUC *usecase.AuthorizeTransactionUseCase,
	getCardUC *usecase.GetCardUseCase,
	freezeCardUC *usecase.FreezeCardUseCase,
) *CardServiceHandler {
	return &CardServiceHandler{
		issueCardUC:  issueCardUC,
		authorizeUC:  authorizeUC,
		getCardUC:    getCardUC,
		freezeCardUC: freezeCardUC,
	}
}

// IssueCard handles the gRPC request to issue a new card.
func (h *CardServiceHandler) IssueCard(ctx context.Context, tenantID, accountID, cardType, currency string, dailyLimit, monthlyLimit string) (*dto.IssueCardResponse, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	daily, err := decimal.NewFromString(dailyLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid daily_limit: %w", err)
	}

	monthly, err := decimal.NewFromString(monthlyLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid monthly_limit: %w", err)
	}

	req := dto.IssueCardRequest{
		TenantID:     tenantUUID,
		AccountID:    accountUUID,
		CardType:     cardType,
		Currency:     currency,
		DailyLimit:   daily,
		MonthlyLimit: monthly,
	}

	resp, err := h.issueCardUC.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AuthorizeTransaction handles the gRPC request to authorize a card transaction.
func (h *CardServiceHandler) AuthorizeTransaction(ctx context.Context, cardID, currency, merchantName, merchantCategory string, amount string) (*dto.AuthorizeTransactionResponse, error) {
	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return nil, fmt.Errorf("invalid card_id: %w", err)
	}

	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	req := dto.AuthorizeTransactionRequest{
		CardID:           cardUUID,
		Amount:           amountDec,
		Currency:         currency,
		MerchantName:     merchantName,
		MerchantCategory: merchantCategory,
	}

	resp, err := h.authorizeUC.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetCard handles the gRPC request to retrieve card details.
func (h *CardServiceHandler) GetCard(ctx context.Context, cardID string) (*dto.CardResponse, error) {
	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return nil, fmt.Errorf("invalid card_id: %w", err)
	}

	req := dto.GetCardRequest{
		CardID: cardUUID,
	}

	resp, err := h.getCardUC.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// FreezeCard handles the gRPC request to freeze a card.
func (h *CardServiceHandler) FreezeCard(ctx context.Context, cardID string) (*dto.FreezeCardResponse, error) {
	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return nil, fmt.Errorf("invalid card_id: %w", err)
	}

	req := dto.FreezeCardRequest{
		CardID: cardUUID,
	}

	resp, err := h.freezeCardUC.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
