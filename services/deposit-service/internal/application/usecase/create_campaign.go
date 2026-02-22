package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

// --- Campaign DTOs ---

// CreateCampaignRequest is the input for creating a deposit campaign.
type CreateCampaignRequest struct {
	TenantID            uuid.UUID
	Name                string
	Description         string
	ProductID           uuid.UUID
	BonusRateBps        int
	EligibilityCriteria string
	MinDeposit          decimal.Decimal
	MaxDeposit          decimal.Decimal
	TargetAudience      string
	StartDate           time.Time
	EndDate             time.Time
}

// CampaignResponse is the output DTO for a campaign.
type CampaignResponse struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	Name              string
	Description       string
	ProductID         uuid.UUID
	BonusRateBps      int
	TargetAudience    string
	StartDate         time.Time
	EndDate           time.Time
	Status            string
	TotalEnrollments  int
	TotalDepositValue string
	Version           int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CreateCampaign handles the creation of new deposit campaigns.
type CreateCampaign struct {
	campaignRepo port.CampaignRepository
	productRepo  port.DepositProductRepository
}

// NewCreateCampaign creates a new CreateCampaign use case.
func NewCreateCampaign(
	campaignRepo port.CampaignRepository,
	productRepo port.DepositProductRepository,
) *CreateCampaign {
	return &CreateCampaign{
		campaignRepo: campaignRepo,
		productRepo:  productRepo,
	}
}

// Execute creates a new deposit campaign.
func (uc *CreateCampaign) Execute(ctx context.Context, req CreateCampaignRequest) (CampaignResponse, error) {
	// Verify the product exists
	_, err := uc.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return CampaignResponse{}, fmt.Errorf("product not found: %w", err)
	}

	// Create promotional rate value object
	promoRate, err := valueobject.NewPromotionalRate(
		req.BonusRateBps,
		req.EligibilityCriteria,
		req.MinDeposit,
		req.MaxDeposit,
	)
	if err != nil {
		return CampaignResponse{}, fmt.Errorf("invalid promotional rate: %w", err)
	}

	// Create campaign aggregate
	campaign, err := model.NewCampaign(
		req.TenantID,
		req.Name,
		req.Description,
		req.ProductID,
		promoRate,
		model.TargetAudience(req.TargetAudience),
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		return CampaignResponse{}, fmt.Errorf("failed to create campaign: %w", err)
	}

	// Persist
	if err := uc.campaignRepo.Save(ctx, campaign); err != nil {
		return CampaignResponse{}, fmt.Errorf("failed to save campaign: %w", err)
	}

	return toCampaignResponse(campaign), nil
}

func toCampaignResponse(c model.Campaign) CampaignResponse {
	return CampaignResponse{
		ID:                c.ID(),
		TenantID:          c.TenantID(),
		Name:              c.Name(),
		Description:       c.Description(),
		ProductID:         c.ProductID(),
		BonusRateBps:      c.PromotionalRate().BonusRateBps(),
		TargetAudience:    string(c.TargetAudience()),
		StartDate:         c.StartDate(),
		EndDate:           c.EndDate(),
		Status:            string(c.Status()),
		TotalEnrollments:  c.TotalEnrollments(),
		TotalDepositValue: c.TotalDepositValue(),
		Version:           c.Version(),
		CreatedAt:         c.CreatedAt(),
		UpdatedAt:         c.UpdatedAt(),
	}
}
