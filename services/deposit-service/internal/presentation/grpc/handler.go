package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// Compile-time assertion that DepositHandler implements DepositServiceServer.
var _ DepositServiceServer = (*DepositHandler)(nil)

// DepositHandler implements the gRPC DepositServiceServer interface.
type DepositHandler struct {
	UnimplementedDepositServiceServer
	createProduct  *usecase.CreateDepositProduct
	openPosition   *usecase.OpenDepositPosition
	getPosition    *usecase.GetDepositPosition
	accrueInterest *usecase.AccrueInterest

	logger               *slog.Logger}

func NewDepositHandler(
	createProduct *usecase.CreateDepositProduct,
	openPosition *usecase.OpenDepositPosition,
	getPosition *usecase.GetDepositPosition,
	accrueInterest *usecase.AccrueInterest,
	logger *slog.Logger,
) *DepositHandler {
	return &DepositHandler{
		createProduct:  createProduct,
		openPosition:   openPosition,
		getPosition:    getPosition,
		accrueInterest: accrueInterest,
	
		logger:               logger,}
}

// Proto-aligned request/response message types.

type CreateDepositProductRequest struct {
	TenantID string             `json:"tenant_id"`
	Name     string             `json:"name"`
	Currency string             `json:"currency"`
	Tiers    []*InterestTierMsg `json:"tiers"`
	TermDays int32              `json:"term_days"`
}

type InterestTierMsg struct {
	MinBalance string `json:"min_balance"`
	MaxBalance string `json:"max_balance"`
	RateBps    int32  `json:"rate_bps"`
}

type DepositProductMsg struct {
	ID        string             `json:"id"`
	TenantID  string             `json:"tenant_id"`
	Name      string             `json:"name"`
	Currency  string             `json:"currency"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
	Tiers     []*InterestTierMsg `json:"tiers"`
	TermDays  int32              `json:"term_days"`
	Version   int32              `json:"version"`
	IsActive  bool               `json:"is_active"`
}

type CreateDepositProductResponse struct {
	Product *DepositProductMsg `json:"product"`
}

type OpenDepositPositionRequest struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	ProductID string `json:"product_id"`
	Principal string `json:"principal"`
}

type DepositPositionMsg struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	AccountID       string `json:"account_id"`
	ProductID       string `json:"product_id"`
	Principal       string `json:"principal"`
	Currency        string `json:"currency"`
	AccruedInterest string `json:"accrued_interest"`
	Status          string `json:"status"`
	OpenedAt        string `json:"opened_at"`
	LastAccrualDate string `json:"last_accrual_date"`
	MaturityDate    string `json:"maturity_date,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Version         int32  `json:"version"`
}

type OpenDepositPositionResponse struct {
	Position *DepositPositionMsg `json:"position"`
}

type GetDepositPositionRequest struct {
	ID string `json:"id"`
}

type GetDepositPositionResponse struct {
	Position *DepositPositionMsg `json:"position"`
}

type AccrueInterestRequest struct {
	AsOfDate string `json:"as_of_date"`
	TenantID string `json:"tenant_id"`
}

type AccrueInterestResponse struct {
	TotalAccrued       string `json:"total_accrued"`
	PositionsProcessed int32  `json:"positions_processed"`
}

// CreateDepositProduct processes product creation requests.
func (h *DepositHandler) CreateDepositProduct(ctx context.Context, req *CreateDepositProductRequest) (*CreateDepositProductResponse, error) {
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

	var tiers []dto.InterestTierDTO
	for _, t := range req.Tiers {
		minBal, parseErr := decimal.NewFromString(t.MinBalance)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid min_balance: %v", parseErr)
		}
		maxBal, parseErr := decimal.NewFromString(t.MaxBalance)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid max_balance: %v", parseErr)
		}
		if minBal.IsNegative() {
			return nil, status.Error(codes.InvalidArgument, "min_balance must not be negative")
		}
		if !maxBal.IsPositive() {
			return nil, status.Error(codes.InvalidArgument, "max_balance must be positive")
		}
		rateBps := int(t.RateBps)
		tiers = append(tiers, dto.InterestTierDTO{
			MinBalance: minBal,
			MaxBalance: maxBal,
			RateBps:    rateBps,
		})
	}

	result, err := h.createProduct.Execute(ctx, dto.CreateDepositProductRequest{
		TenantID: tenantID,
		Name:     req.Name,
		Currency: req.Currency,
		Tiers:    tiers,
		TermDays: int(req.TermDays),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &CreateDepositProductResponse{
		Product: toDepositProductMsg(result),
	}, nil
}

// OpenDepositPosition processes position opening requests.
func (h *DepositHandler) OpenDepositPosition(ctx context.Context, req *OpenDepositPositionRequest) (*OpenDepositPositionResponse, error) {
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

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account_id: %v", err)
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product_id: %v", err)
	}
	principal, err := decimal.NewFromString(req.Principal)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid principal: %v", err)
	}
	if !principal.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "principal must be positive")
	}

	result, err := h.openPosition.Execute(ctx, dto.OpenPositionRequest{
		TenantID:  tenantID,
		AccountID: accountID,
		ProductID: productID,
		Principal: principal,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &OpenDepositPositionResponse{
		Position: toPositionMsg(result),
	}, nil
}

// GetDepositPosition processes position retrieval requests.
func (h *DepositHandler) GetDepositPosition(ctx context.Context, req *GetDepositPositionRequest) (*GetDepositPositionResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	positionID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	result, err := h.getPosition.Execute(ctx, dto.GetPositionRequest{
		PositionID: positionID,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "position not found: %v", err)
	}

	return &GetDepositPositionResponse{
		Position: toPositionMsg(result),
	}, nil
}

// AccrueInterest processes batch interest accrual requests.
func (h *DepositHandler) AccrueInterest(ctx context.Context, req *AccrueInterestRequest) (*AccrueInterestResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var asOf time.Time
	if req.AsOfDate != "" {
		var parseErr error
		asOf, parseErr = time.Parse(time.RFC3339, req.AsOfDate)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid as_of_date: %v", parseErr)
		}
	} else {
		asOf = time.Now()
	}

	result, err := h.accrueInterest.Execute(ctx, dto.AccrueInterestRequest{
		TenantID: tenantID,
		AsOf:     asOf,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &AccrueInterestResponse{
		PositionsProcessed: int32(result.PositionsProcessed), //nolint:gosec
		TotalAccrued:       result.TotalAccrued.String(),
	}, nil
}

func toDepositProductMsg(r dto.DepositProductResponse) *DepositProductMsg {
	var tiers []*InterestTierMsg
	for _, t := range r.Tiers {
		tiers = append(tiers, &InterestTierMsg{
			MinBalance: t.MinBalance.String(),
			MaxBalance: t.MaxBalance.String(),
			RateBps:    int32(t.RateBps), //nolint:gosec
		})
	}
	return &DepositProductMsg{
		ID:        r.ID.String(),
		TenantID:  r.TenantID.String(),
		Name:      r.Name,
		Currency:  r.Currency,
		Tiers:     tiers,
		TermDays:  int32(r.TermDays), //nolint:gosec
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
		UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
		Version:   int32(r.Version), //nolint:gosec
		IsActive:  r.IsActive,
	}
}

func toPositionMsg(r dto.DepositPositionResponse) *DepositPositionMsg {
	msg := &DepositPositionMsg{
		ID:              r.ID.String(),
		TenantID:        r.TenantID.String(),
		AccountID:       r.AccountID.String(),
		ProductID:       r.ProductID.String(),
		Principal:       r.Principal.String(),
		Currency:        r.Currency,
		AccruedInterest: r.AccruedInterest.String(),
		Status:          r.Status,
		OpenedAt:        r.OpenedAt.Format(time.RFC3339),
		LastAccrualDate: r.LastAccrualDate.Format(time.RFC3339),
		CreatedAt:       r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.Format(time.RFC3339),
		Version:         int32(r.Version), //nolint:gosec
	}
	if r.MaturityDate != nil {
		msg.MaturityDate = r.MaturityDate.Format(time.RFC3339)
	}
	return msg
}
