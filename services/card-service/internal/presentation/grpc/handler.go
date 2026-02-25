package grpc

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
)

var currencyCodeRE = regexp.MustCompile(`^[A-Z]{3}$`)

// requireRole checks that the caller has at least one of the given roles.
func requireRole(ctx context.Context, roles ...string) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	for _, role := range roles {
		if claims.HasRole(role) {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "insufficient permissions")
}

// tenantIDFromContext extracts the tenant ID from JWT claims in the context.
func tenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return claims.TenantID, nil
}

// Compile-time assertion that CardServiceHandler implements CardServiceServer.
var _ CardServiceServer = (*CardServiceHandler)(nil)

// CardServiceHandler implements the gRPC CardServiceServer interface.
type CardServiceHandler struct {
	UnimplementedCardServiceServer
	issueCardUC  *usecase.IssueCardUseCase
	authorizeUC  *usecase.AuthorizeTransactionUseCase
	getCardUC    *usecase.GetCardUseCase
	freezeCardUC *usecase.FreezeCardUseCase
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

// Proto-aligned request/response message types.

// IssueCardRequest represents the proto IssueCardRequest message.
type IssueCardRequest struct {
	TenantID     string `json:"tenant_id"`
	AccountID    string `json:"account_id"`
	CardType     string `json:"card_type"`
	Currency     string `json:"currency"`
	DailyLimit   string `json:"daily_limit"`
	MonthlyLimit string `json:"monthly_limit"`
}

// IssueCardResponse represents the proto IssueCardResponse message.
type IssueCardResponse struct {
	CardID string `json:"card_id"`
	Status string `json:"status"`
}

// AuthorizeTransactionRequest represents the proto AuthorizeTransactionRequest message.
type AuthorizeTransactionRequest struct {
	CardID           string `json:"card_id"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	MerchantName     string `json:"merchant_name"`
	MerchantCategory string `json:"merchant_category"`
}

// AuthorizeTransactionResponse represents the proto AuthorizeTransactionResponse message.
type AuthorizeTransactionResponse struct {
	DeclineReason     string `json:"decline_reason"`
	AuthorizationCode string `json:"authorization_code"`
	Approved          bool   `json:"approved"`
}

// GetCardRequest represents the proto GetCardRequest message.
type GetCardRequest struct {
	ID string `json:"card_id"`
}

// GetCardResponse represents the proto GetCardResponse message.
type GetCardResponse struct {
	CardID       string `json:"card_id"`
	TenantID     string `json:"tenant_id"`
	AccountID    string `json:"account_id"`
	CardType     string `json:"card_type"`
	Status       string `json:"status"`
	Currency     string `json:"currency"`
	DailyLimit   string `json:"daily_limit"`
	MonthlyLimit string `json:"monthly_limit"`
	MaskedPan    string `json:"masked_pan"`
	Version      int32  `json:"version"`
}

// IssueCard handles the gRPC request to issue a new card.
func (h *CardServiceHandler) IssueCard(ctx context.Context, req *IssueCardRequest) (*IssueCardResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountUUID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account_id: %v", err)
	}

	if req.CardType == "" {
		return nil, status.Error(codes.InvalidArgument, "card_type is required")
	}

	var dailyLimit, monthlyLimit decimal.Decimal

	if req.DailyLimit != "" {
		dailyLimit, err = decimal.NewFromString(req.DailyLimit)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid daily_limit amount: %v", err)
		}
		if !dailyLimit.IsPositive() {
			return nil, status.Error(codes.InvalidArgument, "daily_limit amount must be positive")
		}
	}

	if req.MonthlyLimit != "" {
		monthlyLimit, err = decimal.NewFromString(req.MonthlyLimit)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid monthly_limit amount: %v", err)
		}
		if !monthlyLimit.IsPositive() {
			return nil, status.Error(codes.InvalidArgument, "monthly_limit amount must be positive")
		}
	}

	currency := req.Currency
	if currency != "" && !currencyCodeRE.MatchString(currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}

	dtoReq := dto.IssueCardRequest{
		TenantID:     tenantID,
		AccountID:    accountUUID,
		CardType:     req.CardType,
		Currency:     currency,
		DailyLimit:   dailyLimit,
		MonthlyLimit: monthlyLimit,
	}

	resp, err := h.issueCardUC.Execute(ctx, dtoReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &IssueCardResponse{
		CardID: resp.CardID.String(),
		Status: resp.Status,
	}, nil
}

// AuthorizeTransaction handles the gRPC request to authorize a card transaction.
func (h *CardServiceHandler) AuthorizeTransaction(ctx context.Context, req *AuthorizeTransactionRequest) (*AuthorizeTransactionResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	cardUUID, err := uuid.Parse(req.CardID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid card_id: %v", err)
	}

	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amount.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	currency := req.Currency
	if currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if !currencyCodeRE.MatchString(currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}
	if req.MerchantName == "" {
		return nil, status.Error(codes.InvalidArgument, "merchant_name is required")
	}

	dtoReq := dto.AuthorizeTransactionRequest{
		CardID:           cardUUID,
		Amount:           amount,
		Currency:         currency,
		MerchantName:     req.MerchantName,
		MerchantCategory: req.MerchantCategory,
	}

	resp, err := h.authorizeUC.Execute(ctx, dtoReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &AuthorizeTransactionResponse{
		Approved:          resp.Approved,
		DeclineReason:     resp.Reason,
		AuthorizationCode: resp.AuthCode,
	}, nil
}

// GetCard handles the gRPC request to retrieve card details.
func (h *CardServiceHandler) GetCard(ctx context.Context, req *GetCardRequest) (*GetCardResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	cardUUID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	resp, err := h.getCardUC.Execute(ctx, dto.GetCardRequest{
		CardID: cardUUID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetCardResponse{
		CardID:       resp.ID.String(),
		TenantID:     resp.TenantID.String(),
		AccountID:    resp.AccountID.String(),
		CardType:     resp.CardType,
		Status:       resp.Status,
		Currency:     resp.Currency,
		DailyLimit:   resp.DailyLimit.String(),
		MonthlyLimit: resp.MonthlyLimit.String(),
		MaskedPan:    resp.LastFour,
		Version:      1,
	}, nil
}

// FreezeCard handles the gRPC request to freeze a card.
func (h *CardServiceHandler) FreezeCard(ctx context.Context, req *FreezeCardGRPCRequest) (*FreezeCardGRPCResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	cardUUID, err := uuid.Parse(req.CardID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid card_id: %v", err)
	}

	resp, err := h.freezeCardUC.Execute(ctx, dto.FreezeCardRequest{
		CardID: cardUUID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &FreezeCardGRPCResponse{
		CardID: resp.CardID.String(),
		Status: resp.Status,
	}, nil
}
