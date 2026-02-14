package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
)

const TopicDepositEvents = "bib.deposit.events"

// OpenDepositPosition handles opening new deposit positions.
type OpenDepositPosition struct {
	productRepo  port.DepositProductRepository
	positionRepo port.DepositPositionRepository
	publisher    port.EventPublisher
}

func NewOpenDepositPosition(
	productRepo port.DepositProductRepository,
	positionRepo port.DepositPositionRepository,
	publisher port.EventPublisher,
) *OpenDepositPosition {
	return &OpenDepositPosition{
		productRepo:  productRepo,
		positionRepo: positionRepo,
		publisher:    publisher,
	}
}

func (uc *OpenDepositPosition) Execute(ctx context.Context, req dto.OpenPositionRequest) (dto.DepositPositionResponse, error) {
	// Validate product exists and is active
	product, err := uc.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return dto.DepositPositionResponse{}, fmt.Errorf("product not found: %w", err)
	}
	if !product.IsActive() {
		return dto.DepositPositionResponse{}, fmt.Errorf("product %s is not active", req.ProductID)
	}

	// Compute maturity date for term deposits
	var maturityDate *time.Time
	if product.IsTermDeposit() {
		md := time.Now().UTC().AddDate(0, 0, product.TermDays())
		maturityDate = &md
	}

	// Create deposit position
	position, err := model.NewDepositPosition(
		req.TenantID,
		req.AccountID,
		req.ProductID,
		req.Principal,
		product.Currency(),
		maturityDate,
	)
	if err != nil {
		return dto.DepositPositionResponse{}, fmt.Errorf("failed to create deposit position: %w", err)
	}

	// Persist
	if err := uc.positionRepo.Save(ctx, position); err != nil {
		return dto.DepositPositionResponse{}, fmt.Errorf("failed to save deposit position: %w", err)
	}

	// Publish domain events
	if events := position.DomainEvents(); len(events) > 0 {
		if err := uc.publisher.Publish(ctx, TopicDepositEvents, events...); err != nil {
			return dto.DepositPositionResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	return toPositionResponse(position), nil
}

func toPositionResponse(p model.DepositPosition) dto.DepositPositionResponse {
	return dto.DepositPositionResponse{
		ID:              p.ID(),
		TenantID:        p.TenantID(),
		AccountID:       p.AccountID(),
		ProductID:       p.ProductID(),
		Principal:       p.Principal(),
		Currency:        p.Currency(),
		AccruedInterest: p.AccruedInterest(),
		Status:          string(p.Status()),
		OpenedAt:        p.OpenedAt(),
		MaturityDate:    p.MaturityDate(),
		LastAccrualDate: p.LastAccrualDate(),
		Version:         p.Version(),
		CreatedAt:       p.CreatedAt(),
		UpdatedAt:       p.UpdatedAt(),
	}
}
