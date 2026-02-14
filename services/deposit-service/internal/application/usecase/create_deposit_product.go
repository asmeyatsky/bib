package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

// CreateDepositProduct handles the creation of new deposit products.
type CreateDepositProduct struct {
	productRepo port.DepositProductRepository
}

func NewCreateDepositProduct(productRepo port.DepositProductRepository) *CreateDepositProduct {
	return &CreateDepositProduct{productRepo: productRepo}
}

func (uc *CreateDepositProduct) Execute(ctx context.Context, req dto.CreateDepositProductRequest) (dto.DepositProductResponse, error) {
	// Convert DTO tiers to domain value objects
	var tiers []valueobject.InterestTier
	for _, t := range req.Tiers {
		tier, err := valueobject.NewInterestTier(t.MinBalance, t.MaxBalance, t.RateBps)
		if err != nil {
			return dto.DepositProductResponse{}, fmt.Errorf("invalid interest tier: %w", err)
		}
		tiers = append(tiers, tier)
	}

	// Create domain aggregate
	product, err := model.NewDepositProduct(req.TenantID, req.Name, req.Currency, tiers, req.TermDays)
	if err != nil {
		return dto.DepositProductResponse{}, fmt.Errorf("failed to create deposit product: %w", err)
	}

	// Persist
	if err := uc.productRepo.Save(ctx, product); err != nil {
		return dto.DepositProductResponse{}, fmt.Errorf("failed to save deposit product: %w", err)
	}

	return toDepositProductResponse(product), nil
}

func toDepositProductResponse(p model.DepositProduct) dto.DepositProductResponse {
	var tiers []dto.InterestTierDTO
	for _, t := range p.Tiers() {
		tiers = append(tiers, dto.InterestTierDTO{
			MinBalance: t.MinBalance(),
			MaxBalance: t.MaxBalance(),
			RateBps:    t.RateBps(),
		})
	}
	return dto.DepositProductResponse{
		ID:        p.ID(),
		TenantID:  p.TenantID(),
		Name:      p.Name(),
		Currency:  p.Currency(),
		Tiers:     tiers,
		TermDays:  p.TermDays(),
		IsActive:  p.IsActive(),
		Version:   p.Version(),
		CreatedAt: p.CreatedAt(),
		UpdatedAt: p.UpdatedAt(),
	}
}
