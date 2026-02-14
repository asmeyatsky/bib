package usecase

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/service"
)

const TopicDepositInterest = "bib.deposit.interest"

// AccrueInterest handles batch interest accrual for all active positions of a tenant.
type AccrueInterest struct {
	productRepo  port.DepositProductRepository
	positionRepo port.DepositPositionRepository
	publisher    port.EventPublisher
	engine       *service.AccrualEngine
}

func NewAccrueInterest(
	productRepo port.DepositProductRepository,
	positionRepo port.DepositPositionRepository,
	publisher port.EventPublisher,
	engine *service.AccrualEngine,
) *AccrueInterest {
	return &AccrueInterest{
		productRepo:  productRepo,
		positionRepo: positionRepo,
		publisher:    publisher,
		engine:       engine,
	}
}

func (uc *AccrueInterest) Execute(ctx context.Context, req dto.AccrueInterestRequest) (dto.AccrueInterestResponse, error) {
	// Fetch all active positions for the tenant
	positions, err := uc.positionRepo.FindActiveByTenant(ctx, req.TenantID)
	if err != nil {
		return dto.AccrueInterestResponse{}, fmt.Errorf("failed to fetch active positions: %w", err)
	}

	// Cache products to avoid repeated lookups
	productCache := make(map[string]interface{})
	_ = productCache // we'll use a simpler approach below

	totalAccrued := decimal.Zero
	processed := 0

	for _, position := range positions {
		// Fetch product for this position
		product, err := uc.productRepo.FindByID(ctx, position.ProductID())
		if err != nil {
			return dto.AccrueInterestResponse{}, fmt.Errorf("failed to fetch product %s: %w", position.ProductID(), err)
		}

		// Accrue interest using the domain service
		accrued, err := uc.engine.AccrueForPosition(position, product, req.AsOf)
		if err != nil {
			return dto.AccrueInterestResponse{}, fmt.Errorf("failed to accrue for position %s: %w", position.ID(), err)
		}

		// Persist the updated position
		if err := uc.positionRepo.Save(ctx, accrued); err != nil {
			return dto.AccrueInterestResponse{}, fmt.Errorf("failed to save position %s: %w", position.ID(), err)
		}

		// Publish interest accrual events to Kafka for ledger to post accrual entries
		if events := accrued.DomainEvents(); len(events) > 0 {
			if err := uc.publisher.Publish(ctx, TopicDepositInterest, events...); err != nil {
				return dto.AccrueInterestResponse{}, fmt.Errorf("failed to publish events for position %s: %w", position.ID(), err)
			}
		}

		// Track the accrued amount (difference from before)
		accruedDiff := accrued.AccruedInterest().Sub(position.AccruedInterest())
		totalAccrued = totalAccrued.Add(accruedDiff)
		processed++
	}

	return dto.AccrueInterestResponse{
		PositionsProcessed: processed,
		TotalAccrued:       totalAccrued,
	}, nil
}
