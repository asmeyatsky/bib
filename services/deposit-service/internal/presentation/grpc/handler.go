package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
)

// DepositHandler implements the gRPC DepositService server.
type DepositHandler struct {
	createProduct *usecase.CreateDepositProduct
	openPosition  *usecase.OpenDepositPosition
	getPosition   *usecase.GetDepositPosition
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

// --- Temporary request/response types until proto generation is wired ---

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
	RateBps    int32
}

type DepositProductMsg struct {
	ID        string
	TenantID  string
	Name      string
	Currency  string
	Tiers     []*InterestTierMsg
	TermDays  int32
	IsActive  bool
	Version   int32
	CreatedAt string
	UpdatedAt string
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
	OpenedAt        string
	MaturityDate    string
	LastAccrualDate string
	Version         int32
	CreatedAt       string
	UpdatedAt       string
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
	AsOfDate string
}

type AccrueInterestResponse struct {
	PositionsProcessed int32
	TotalAccrued       string
}

// HandleCreateDepositProduct processes product creation requests.
func (h *DepositHandler) HandleCreateDepositProduct(ctx context.Context, req *CreateDepositProductRequest) (*CreateDepositProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	var tiers []dto.InterestTierDTO
	for _, t := range req.Tiers {
		minBal, err := decimal.NewFromString(t.MinBalance)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid min_balance: %v", err)
		}
		maxBal, err := decimal.NewFromString(t.MaxBalance)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid max_balance: %v", err)
		}
		tiers = append(tiers, dto.InterestTierDTO{
			MinBalance: minBal,
			MaxBalance: maxBal,
			RateBps:    int(t.RateBps),
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
		return nil, status.Errorf(codes.Internal, "failed to create deposit product: %v", err)
	}

	return &CreateDepositProductResponse{
		Product: toDepositProductMsg(result),
	}, nil
}

// HandleOpenDepositPosition processes position opening requests.
func (h *DepositHandler) HandleOpenDepositPosition(ctx context.Context, req *OpenDepositPositionRequest) (*OpenDepositPositionResponse, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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

	result, err := h.openPosition.Execute(ctx, dto.OpenPositionRequest{
		TenantID:  tenantID,
		AccountID: accountID,
		ProductID: productID,
		Principal: principal,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to open deposit position: %v", err)
	}

	return &OpenDepositPositionResponse{
		Position: toPositionMsg(result),
	}, nil
}

// HandleGetDepositPosition processes position retrieval requests.
func (h *DepositHandler) HandleGetDepositPosition(ctx context.Context, req *GetDepositPositionRequest) (*GetDepositPositionResponse, error) {
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

// HandleAccrueInterest processes batch interest accrual requests.
func (h *DepositHandler) HandleAccrueInterest(ctx context.Context, req *AccrueInterestRequest) (*AccrueInterestResponse, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	asOf, err := time.Parse("2006-01-02", req.AsOfDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid as_of_date: %v", err)
	}

	result, err := h.accrueInterest.Execute(ctx, dto.AccrueInterestRequest{
		TenantID: tenantID,
		AsOf:     asOf,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to accrue interest: %v", err)
	}

	return &AccrueInterestResponse{
		PositionsProcessed: int32(result.PositionsProcessed),
		TotalAccrued:       result.TotalAccrued.String(),
	}, nil
}

func toDepositProductMsg(r dto.DepositProductResponse) *DepositProductMsg {
	var tiers []*InterestTierMsg
	for _, t := range r.Tiers {
		tiers = append(tiers, &InterestTierMsg{
			MinBalance: t.MinBalance.String(),
			MaxBalance: t.MaxBalance.String(),
			RateBps:    int32(t.RateBps),
		})
	}
	return &DepositProductMsg{
		ID:        r.ID.String(),
		TenantID:  r.TenantID.String(),
		Name:      r.Name,
		Currency:  r.Currency,
		Tiers:     tiers,
		TermDays:  int32(r.TermDays),
		IsActive:  r.IsActive,
		Version:   int32(r.Version),
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
		UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
	}
}

func toPositionMsg(r dto.DepositPositionResponse) *DepositPositionMsg {
	var maturityDate string
	if r.MaturityDate != nil {
		maturityDate = r.MaturityDate.Format(time.RFC3339)
	}
	return &DepositPositionMsg{
		ID:              r.ID.String(),
		TenantID:        r.TenantID.String(),
		AccountID:       r.AccountID.String(),
		ProductID:       r.ProductID.String(),
		Principal:       r.Principal.String(),
		Currency:        r.Currency,
		AccruedInterest: r.AccruedInterest.String(),
		Status:          r.Status,
		OpenedAt:        r.OpenedAt.Format(time.RFC3339),
		MaturityDate:    maturityDate,
		LastAccrualDate: r.LastAccrualDate.Format(time.RFC3339),
		Version:         int32(r.Version),
		CreatedAt:       r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.Format(time.RFC3339),
	}
}
