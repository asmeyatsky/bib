package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/domain/event"
	"github.com/bibbank/bib/services/fx-service/internal/domain/port"
	"github.com/bibbank/bib/services/fx-service/internal/domain/service"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

const TopicFXRevaluation = "bib.fx.revaluation"

// Revaluate runs an ASC 830 FX revaluation for the given positions and publishes
// the result as a domain event.
type Revaluate struct {
	rateRepo    port.ExchangeRateRepository
	publisher   port.EventPublisher
	revalEngine *service.RevaluationEngine
}

// NewRevaluate creates a new Revaluate use case.
func NewRevaluate(
	rateRepo port.ExchangeRateRepository,
	publisher port.EventPublisher,
	revalEngine *service.RevaluationEngine,
) *Revaluate {
	return &Revaluate{
		rateRepo:    rateRepo,
		publisher:   publisher,
		revalEngine: revalEngine,
	}
}

// Execute runs the revaluation engine across all provided positions.
func (uc *Revaluate) Execute(ctx context.Context, req dto.RevaluateRequest) (dto.RevaluateResponse, error) {
	if req.TenantID == uuid.Nil {
		return dto.RevaluateResponse{}, fmt.Errorf("tenant ID is required")
	}
	if req.FunctionalCurrency == "" {
		return dto.RevaluateResponse{}, fmt.Errorf("functional currency is required")
	}
	if len(req.Positions) == 0 {
		return dto.RevaluateResponse{}, fmt.Errorf("at least one position is required")
	}

	// Collect unique foreign currencies and build rate lookup.
	currencySet := make(map[string]struct{})
	for _, pos := range req.Positions {
		if pos.Currency != req.FunctionalCurrency {
			currencySet[pos.Currency] = struct{}{}
		}
	}

	rates := make(map[string]valueobject.SpotRate)
	for currency := range currencySet {
		pair, err := valueobject.NewCurrencyPair(currency, req.FunctionalCurrency)
		if err != nil {
			return dto.RevaluateResponse{}, fmt.Errorf("invalid pair %s/%s: %w", currency, req.FunctionalCurrency, err)
		}

		er, err := uc.rateRepo.FindByPair(ctx, req.TenantID, pair)
		if err != nil {
			return dto.RevaluateResponse{}, fmt.Errorf("rate not found for %s: %w", pair.String(), err)
		}

		rates[currency] = er.Rate()
	}

	// Convert DTO positions to domain positions.
	positions := make([]service.ForeignCurrencyPosition, 0, len(req.Positions))
	for _, p := range req.Positions {
		positions = append(positions, service.ForeignCurrencyPosition{
			AccountCode: p.AccountCode,
			Currency:    p.Currency,
			Amount:      p.Amount,
		})
	}

	// Run the revaluation engine.
	entries, totalGainLoss := uc.revalEngine.Revaluate(positions, rates, req.FunctionalCurrency)

	// Build response DTOs.
	entryDTOs := make([]dto.RevaluationEntryDTO, 0, len(entries))
	for _, e := range entries {
		entryDTOs = append(entryDTOs, dto.RevaluationEntryDTO{
			AccountCode:        e.AccountCode(),
			OriginalCurrency:   e.OriginalCurrency(),
			FunctionalCurrency: e.FunctionalCurrency(),
			OriginalAmount:     e.OriginalAmount(),
			RevaluedAmount:     e.RevaluedAmount(),
			GainLoss:           e.GainLoss(),
			Rate:               e.Rate().Rate(),
		})
	}

	// Publish RevaluationCompleted event.
	evt := event.NewRevaluationCompleted(
		req.TenantID,
		req.FunctionalCurrency,
		totalGainLoss.StringFixed(4),
		len(entries),
	)
	if err := uc.publisher.Publish(ctx, TopicFXRevaluation, evt); err != nil {
		return dto.RevaluateResponse{}, fmt.Errorf("publish revaluation event: %w", err)
	}

	return dto.RevaluateResponse{
		TenantID:           req.TenantID,
		FunctionalCurrency: req.FunctionalCurrency,
		TotalGainLoss:      totalGainLoss,
		Entries:            entryDTOs,
	}, nil
}
