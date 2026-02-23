package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
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
}

func NewDepositHandler(
	createProduct *usecase.CreateDepositProduct,
	openPosition *usecase.OpenDepositPosition,
	getPosition *usecase.GetDepositPosition,
	accrueInterest *usecase.AccrueInterest,
) *DepositHandler {
	return &DepositHandler{
		createProduct:  createProduct,
		openPosition:   openPosition,
		getPosition:    getPosition,
		accrueInterest: accrueInterest,
	}
}

// Proto-aligned request/response message types.

type CreateDepositProductRequest struct {
	TenantID string
	Name     string
	Currency string
	Tiers    []*InterestTierMsg
	TermDays int32
}

type InterestTierMsg struct {
	MinBalance string
	MaxBalance string
	RateBps    string
}

type DepositProductMsg struct {
	ID       string
	TenantID string
	Name     string
	Currency string
	Tiers    []*InterestTierMsg
	TermDays int32
}

type CreateDepositProductResponse struct {
	Product *DepositProductMsg
}

type OpenDepositPositionRequest struct {
	TenantID  string
	AccountID string
	ProductID string
	Principal string
}

type DepositPositionMsg struct {
	ID              string
	TenantID        string
	AccountID       string
	ProductID       string
	Principal       string
	Currency        string
	AccruedInterest string
	Status          string
	OpenedAt        *timestamppb.Timestamp
	MaturityDate    *timestamppb.Timestamp
}

type OpenDepositPositionResponse struct {
	Position *DepositPositionMsg
}

type GetDepositPositionRequest struct {
	ID string
}

type GetDepositPositionResponse struct {
	Position *DepositPositionMsg
}

type AccrueInterestRequest struct {
	TenantID string
	AsOfDate *timestamppb.Timestamp
}

type AccrueInterestResponse struct {
	PositionsProcessed int32
	TotalAccrued       string
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
		rateBps := 0
		if t.RateBps != "" {
			d, parseErr := decimal.NewFromString(t.RateBps)
			if parseErr != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid rate_bps: %v", parseErr)
			}
			rateBps = int(d.IntPart())
		}
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
	if req.AsOfDate != nil {
		asOf = req.AsOfDate.AsTime()
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
			RateBps:    decimal.NewFromInt(int64(t.RateBps)).String(),
		})
	}
	return &DepositProductMsg{
		ID:       r.ID.String(),
		TenantID: r.TenantID.String(),
		Name:     r.Name,
		Currency: r.Currency,
		Tiers:    tiers,
		TermDays: int32(r.TermDays), //nolint:gosec
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
		OpenedAt:        timestamppb.New(r.OpenedAt),
	}
	if r.MaturityDate != nil {
		msg.MaturityDate = timestamppb.New(*r.MaturityDate)
	}
	return msg
}
