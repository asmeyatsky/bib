package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
)

// ApplyCampaignRateRequest is the input for applying a campaign rate to a deposit position.
type ApplyCampaignRateRequest struct {
	PositionID uuid.UUID
	CampaignID uuid.UUID
	AsOf       time.Time
}

// ApplyCampaignRateResponse is the output DTO for the campaign rate application.
type ApplyCampaignRateResponse struct {
	PositionID      uuid.UUID
	CampaignID      uuid.UUID
	BonusInterest   decimal.Decimal
	StandardRate    int // bps
	BonusRate       int // bps
	EffectiveRate   int // bps (standard + bonus)
}

// ApplyCampaignRate applies a promotional campaign rate to a deposit position.
type ApplyCampaignRate struct {
	positionRepo port.DepositPositionRepository
	campaignRepo port.CampaignRepository
	productRepo  port.DepositProductRepository
}

// NewApplyCampaignRate creates a new ApplyCampaignRate use case.
func NewApplyCampaignRate(
	positionRepo port.DepositPositionRepository,
	campaignRepo port.CampaignRepository,
	productRepo port.DepositProductRepository,
) *ApplyCampaignRate {
	return &ApplyCampaignRate{
		positionRepo: positionRepo,
		campaignRepo: campaignRepo,
		productRepo:  productRepo,
	}
}

// Execute applies the campaign's promotional rate to a deposit position,
// accruing the bonus interest on top of the standard rate.
func (uc *ApplyCampaignRate) Execute(ctx context.Context, req ApplyCampaignRateRequest) (ApplyCampaignRateResponse, error) {
	// Fetch position
	position, err := uc.positionRepo.FindByID(ctx, req.PositionID)
	if err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("position not found: %w", err)
	}
	if position.Status() != model.PositionStatusActive {
		return ApplyCampaignRateResponse{}, fmt.Errorf("position is not active")
	}

	// Fetch campaign
	campaign, err := uc.campaignRepo.FindByID(ctx, req.CampaignID)
	if err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("campaign not found: %w", err)
	}
	if !campaign.IsActiveAt(req.AsOf) {
		return ApplyCampaignRateResponse{}, fmt.Errorf("campaign is not active at %s", req.AsOf)
	}
	if campaign.ProductID() != position.ProductID() {
		return ApplyCampaignRateResponse{}, fmt.Errorf("campaign product %s does not match position product %s",
			campaign.ProductID(), position.ProductID())
	}

	// Check deposit eligibility
	promoRate := campaign.PromotionalRate()
	if !promoRate.IsEligible(position.Principal()) {
		return ApplyCampaignRateResponse{}, fmt.Errorf("deposit amount %s not eligible for campaign",
			position.Principal())
	}

	// Fetch product to get standard rate
	product, err := uc.productRepo.FindByID(ctx, position.ProductID())
	if err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("product not found: %w", err)
	}
	tier, err := product.FindApplicableTier(position.TotalBalance())
	if err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("no applicable tier: %w", err)
	}

	// Calculate bonus interest using the promotional daily rate
	bonusDailyRate := promoRate.BonusDailyRate()
	days := daysBetweenAccrual(position.LastAccrualDate(), req.AsOf)
	if days <= 0 {
		return ApplyCampaignRateResponse{}, fmt.Errorf("no days to accrue since last accrual")
	}

	bonusInterest := position.Principal().
		Mul(bonusDailyRate).
		Mul(decimal.NewFromInt(int64(days))).
		Round(4)

	// Apply the bonus interest by accruing with the combined rate
	combinedDailyRate := tier.DailyRate().Add(bonusDailyRate)
	updatedPosition, err := position.AccrueInterest(combinedDailyRate, req.AsOf)
	if err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("failed to accrue interest: %w", err)
	}

	// Persist updated position
	if err := uc.positionRepo.Save(ctx, updatedPosition); err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("failed to save position: %w", err)
	}

	// Record enrollment on campaign
	updatedCampaign := campaign.RecordEnrollment(position.Principal().String(), req.AsOf)
	if err := uc.campaignRepo.Save(ctx, updatedCampaign); err != nil {
		return ApplyCampaignRateResponse{}, fmt.Errorf("failed to update campaign: %w", err)
	}

	return ApplyCampaignRateResponse{
		PositionID:    req.PositionID,
		CampaignID:    req.CampaignID,
		BonusInterest: bonusInterest,
		StandardRate:  tier.RateBps(),
		BonusRate:     promoRate.BonusRateBps(),
		EffectiveRate: tier.RateBps() + promoRate.BonusRateBps(),
	}, nil
}

// daysBetweenAccrual calculates calendar days between two times.
func daysBetweenAccrual(from, to time.Time) int {
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toDate.Sub(fromDate).Hours() / 24)
}
