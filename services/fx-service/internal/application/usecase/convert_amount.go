package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/domain/port"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// ConvertAmount converts an amount from one currency to another using the
// current exchange rate.
type ConvertAmount struct {
	rateRepo     port.ExchangeRateRepository
	rateProvider port.RateProvider
}

// NewConvertAmount creates a new ConvertAmount use case.
func NewConvertAmount(
	rateRepo port.ExchangeRateRepository,
	rateProvider port.RateProvider,
) *ConvertAmount {
	return &ConvertAmount{
		rateRepo:     rateRepo,
		rateProvider: rateProvider,
	}
}

// Execute performs the currency conversion.
func (uc *ConvertAmount) Execute(ctx context.Context, req dto.ConvertAmountRequest) (dto.ConvertAmountResponse, error) {
	if req.Amount.IsNegative() {
		return dto.ConvertAmountResponse{}, fmt.Errorf("amount must not be negative")
	}

	pair, err := valueobject.NewCurrencyPair(req.FromCurrency, req.ToCurrency)
	if err != nil {
		return dto.ConvertAmountResponse{}, fmt.Errorf("invalid currency pair: %w", err)
	}

	// Try to load the rate from the repository.
	existing, err := uc.rateRepo.FindByPair(ctx, req.TenantID, pair)
	if err == nil && !existing.IsExpired(time.Now().UTC()) {
		converted := existing.Convert(req.Amount)
		return dto.ConvertAmountResponse{
			FromCurrency:    req.FromCurrency,
			ToCurrency:      req.ToCurrency,
			OriginalAmount:  req.Amount,
			ConvertedAmount: converted,
			Rate:            existing.Rate().Rate(),
			InverseRate:     existing.InverseRate().Rate(),
			Provider:        existing.Provider(),
			EffectiveAt:     existing.EffectiveAt(),
		}, nil
	}

	// Fallback to external provider.
	spotRate, err := uc.rateProvider.FetchRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		return dto.ConvertAmountResponse{}, fmt.Errorf("fetch rate from provider: %w", err)
	}

	converted := spotRate.Convert(req.Amount)
	now := time.Now().UTC()

	return dto.ConvertAmountResponse{
		FromCurrency:    req.FromCurrency,
		ToCurrency:      req.ToCurrency,
		OriginalAmount:  req.Amount,
		ConvertedAmount: converted,
		Rate:            spotRate.Rate(),
		InverseRate:     spotRate.Inverse().Rate(),
		Provider:        "external-provider",
		EffectiveAt:     now,
	}, nil
}
