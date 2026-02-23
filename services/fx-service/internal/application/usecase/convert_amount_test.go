package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockExchangeRateRepository struct {
	findByPairFunc func(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error)
}

func (m *mockExchangeRateRepository) Save(_ context.Context, _ model.ExchangeRate) error {
	return nil
}

func (m *mockExchangeRateRepository) FindByPair(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error) {
	if m.findByPairFunc != nil {
		return m.findByPairFunc(ctx, tenantID, pair)
	}
	return model.ExchangeRate{}, fmt.Errorf("rate not found")
}

func (m *mockExchangeRateRepository) FindLatest(_ context.Context, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
	return model.ExchangeRate{}, fmt.Errorf("not implemented")
}

func (m *mockExchangeRateRepository) ListByBase(_ context.Context, _ uuid.UUID, _ string, _ time.Time) ([]model.ExchangeRate, error) {
	return nil, nil
}

// --- Tests ---

func TestConvertAmount_Execute(t *testing.T) {
	t.Run("uses cached rate from repository", func(t *testing.T) {
		tenantID := uuid.New()
		pair, _ := valueobject.NewCurrencyPair("USD", "EUR")
		spotRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.92))

		now := time.Now().UTC()
		exRate, _ := model.NewExchangeRate(
			tenantID, pair, spotRate, "reuters",
			now.Add(-time.Minute),
			now.Add(time.Hour), // not expired
		)

		rateRepo := &mockExchangeRateRepository{
			findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
				return exRate, nil
			},
		}
		provider := &mockRateProvider{}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     tenantID,
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Amount:       decimal.NewFromInt(1000),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.FromCurrency)
		assert.Equal(t, "EUR", resp.ToCurrency)
		assert.True(t, decimal.NewFromInt(1000).Equal(resp.OriginalAmount))
		assert.True(t, resp.ConvertedAmount.GreaterThan(decimal.Zero))
		assert.Equal(t, "reuters", resp.Provider)
	})

	t.Run("falls back to external provider when rate expired", func(t *testing.T) {
		tenantID := uuid.New()
		pair, _ := valueobject.NewCurrencyPair("USD", "EUR")
		spotRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.92))

		now := time.Now().UTC()
		expiredRate, _ := model.NewExchangeRate(
			tenantID, pair, spotRate, "reuters",
			now.Add(-2*time.Hour),
			now.Add(-time.Hour), // expired
		)

		rateRepo := &mockExchangeRateRepository{
			findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
				return expiredRate, nil
			},
		}

		providerRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.93))
		provider := &mockRateProvider{
			fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
				return providerRate, nil
			},
		}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     tenantID,
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Amount:       decimal.NewFromInt(1000),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "external-provider", resp.Provider)
		// 1000 * 0.93 = 930
		expected := decimal.NewFromInt(1000).Mul(decimal.NewFromFloat(0.93))
		assert.True(t, expected.Equal(resp.ConvertedAmount))
	})

	t.Run("falls back to external provider when rate not in repo", func(t *testing.T) {
		rateRepo := &mockExchangeRateRepository{
			findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
				return model.ExchangeRate{}, fmt.Errorf("rate not found")
			},
		}

		providerRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(1.25))
		provider := &mockRateProvider{
			fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
				return providerRate, nil
			},
		}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     uuid.New(),
			FromCurrency: "USD",
			ToCurrency:   "GBP",
			Amount:       decimal.NewFromInt(500),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "external-provider", resp.Provider)
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		rateRepo := &mockExchangeRateRepository{}
		provider := &mockRateProvider{}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     uuid.New(),
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Amount:       decimal.NewFromInt(-100),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "amount must not be negative")
	})

	t.Run("fails with invalid currency pair", func(t *testing.T) {
		rateRepo := &mockExchangeRateRepository{}
		provider := &mockRateProvider{}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     uuid.New(),
			FromCurrency: "USD",
			ToCurrency:   "USD", // same currency
			Amount:       decimal.NewFromInt(100),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid currency pair")
	})

	t.Run("fails when external provider is unavailable", func(t *testing.T) {
		rateRepo := &mockExchangeRateRepository{
			findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
				return model.ExchangeRate{}, fmt.Errorf("rate not found")
			},
		}
		provider := &mockRateProvider{
			fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
				return valueobject.SpotRate{}, fmt.Errorf("provider unavailable")
			},
		}

		uc := usecase.NewConvertAmount(rateRepo, provider)

		req := dto.ConvertAmountRequest{
			TenantID:     uuid.New(),
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Amount:       decimal.NewFromInt(100),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch rate from provider")
	})
}
