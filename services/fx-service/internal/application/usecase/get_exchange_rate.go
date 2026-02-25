package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/port"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

const TopicFXRates = "bib.fx.rates"

// GetExchangeRate fetches the current exchange rate for a currency pair.
// If the cached rate is expired, it falls back to the external rate provider
// and persists the refreshed rate.
type GetExchangeRate struct {
	rateRepo     port.ExchangeRateRepository
	rateProvider port.RateProvider
	publisher    port.EventPublisher
}

// NewGetExchangeRate creates a new GetExchangeRate use case.
func NewGetExchangeRate(
	rateRepo port.ExchangeRateRepository,
	rateProvider port.RateProvider,
	publisher port.EventPublisher,
) *GetExchangeRate {
	return &GetExchangeRate{
		rateRepo:     rateRepo,
		rateProvider: rateProvider,
		publisher:    publisher,
	}
}

// Execute retrieves the exchange rate for the requested pair.
func (uc *GetExchangeRate) Execute(ctx context.Context, req dto.GetExchangeRateRequest) (dto.ExchangeRateResponse, error) {
	pair, err := valueobject.NewCurrencyPair(req.BaseCurrency, req.QuoteCurrency)
	if err != nil {
		return dto.ExchangeRateResponse{}, fmt.Errorf("invalid currency pair: %w", err)
	}

	// Try to load the cached rate from the repository.
	existing, err := uc.rateRepo.FindByPair(ctx, req.TenantID, pair)
	if err == nil && !existing.IsExpired(time.Now().UTC()) {
		return toExchangeRateResponse(existing), nil
	}

	// Rate provider is not configured - return cached rate if available.
	if uc.rateProvider == nil {
		if err == nil && existing.ID() != [16]byte{} {
			// Return stale rate with a warning.
			return toExchangeRateResponse(existing), nil
		}
		return dto.ExchangeRateResponse{}, fmt.Errorf("rate provider not configured and no cached rate available")
	}

	// Rate not found or expired - fetch from external provider.
	spotRate, err := uc.rateProvider.FetchRate(ctx, req.BaseCurrency, req.QuoteCurrency)
	if err != nil {
		return dto.ExchangeRateResponse{}, fmt.Errorf("fetch rate from provider: %w", err)
	}

	now := time.Now().UTC()

	// If we have an existing rate, update it; otherwise create a new one.
	var rate model.ExchangeRate
	if existing.ID() != [16]byte{} {
		rate, err = existing.Update(spotRate, "external-provider", now)
		if err != nil {
			return dto.ExchangeRateResponse{}, fmt.Errorf("update rate: %w", err)
		}
	} else {
		rate, err = model.NewExchangeRate(
			req.TenantID,
			pair,
			spotRate,
			"external-provider",
			now,
			now.Add(1*time.Hour),
		)
		if err != nil {
			return dto.ExchangeRateResponse{}, fmt.Errorf("create rate: %w", err)
		}
	}

	// Persist the updated rate.
	if err := uc.rateRepo.Save(ctx, rate); err != nil {
		return dto.ExchangeRateResponse{}, fmt.Errorf("save rate: %w", err)
	}

	// Publish domain events.
	if evts := rate.DomainEvents(); len(evts) > 0 {
		if err := uc.publisher.Publish(ctx, TopicFXRates, evts...); err != nil {
			return dto.ExchangeRateResponse{}, fmt.Errorf("publish events: %w", err)
		}
	}

	return toExchangeRateResponse(rate), nil
}

func toExchangeRateResponse(rate model.ExchangeRate) dto.ExchangeRateResponse {
	return dto.ExchangeRateResponse{
		ID:            rate.ID(),
		TenantID:      rate.TenantID(),
		BaseCurrency:  rate.Pair().Base(),
		QuoteCurrency: rate.Pair().Quote(),
		Rate:          rate.Rate().Rate(),
		InverseRate:   rate.InverseRate().Rate(),
		Provider:      rate.Provider(),
		EffectiveAt:   rate.EffectiveAt(),
		ExpiresAt:     rate.ExpiresAt(),
		Version:       rate.Version(),
		CreatedAt:     rate.CreatedAt(),
	}
}
