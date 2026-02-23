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

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockExchangeRateRepo struct {
	findByPairFunc func(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error)
	saveFunc       func(ctx context.Context, rate model.ExchangeRate) error
	savedRates     []model.ExchangeRate
}

func (m *mockExchangeRateRepo) Save(ctx context.Context, rate model.ExchangeRate) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, rate)
	}
	m.savedRates = append(m.savedRates, rate)
	return nil
}

func (m *mockExchangeRateRepo) FindByPair(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error) {
	if m.findByPairFunc != nil {
		return m.findByPairFunc(ctx, tenantID, pair)
	}
	return model.ExchangeRate{}, fmt.Errorf("not found")
}

func (m *mockExchangeRateRepo) FindLatest(_ context.Context, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
	return model.ExchangeRate{}, fmt.Errorf("not found")
}

func (m *mockExchangeRateRepo) ListByBase(_ context.Context, _ uuid.UUID, _ string, _ time.Time) ([]model.ExchangeRate, error) {
	return nil, nil
}

type mockRateProvider struct {
	fetchRateFunc func(ctx context.Context, base, quote string) (valueobject.SpotRate, error)
}

func (m *mockRateProvider) FetchRate(ctx context.Context, base, quote string) (valueobject.SpotRate, error) {
	if m.fetchRateFunc != nil {
		return m.fetchRateFunc(ctx, base, quote)
	}
	return valueobject.SpotRate{}, fmt.Errorf("provider not configured")
}

type mockEventPublisher struct {
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
	publishedEvents []events.DomainEvent
}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

// --- Tests ---

func TestGetExchangeRate_CachedValid(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	pair, _ := valueobject.NewCurrencyPair("USD", "EUR")
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.85))

	existingRate, _ := model.NewExchangeRate(tenantID, pair, rate, "reuters", now, now.Add(time.Hour))

	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return existingRate, nil
		},
	}
	provider := &mockRateProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	resp, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      tenantID,
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", resp.BaseCurrency)
	assert.Equal(t, "EUR", resp.QuoteCurrency)
	assert.True(t, decimal.NewFromFloat(0.85).Equal(resp.Rate))
	assert.Equal(t, "reuters", resp.Provider)

	// Should not have saved a new rate.
	assert.Empty(t, repo.savedRates)
	// Should not have published events.
	assert.Empty(t, publisher.publishedEvents)
}

func TestGetExchangeRate_ExpiredFallbackToProvider(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	pair, _ := valueobject.NewCurrencyPair("USD", "EUR")
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.85))

	// Expired rate.
	expiredRate, _ := model.NewExchangeRate(tenantID, pair, rate, "reuters",
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return expiredRate, nil
		},
	}

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.87))
	provider := &mockRateProvider{
		fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
			return newRate, nil
		},
	}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	resp, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      tenantID,
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(0.87).Equal(resp.Rate))
	assert.Equal(t, "external-provider", resp.Provider)

	// Should have saved the refreshed rate.
	require.Len(t, repo.savedRates, 1)
}

func TestGetExchangeRate_NotFoundFallbackToProvider(t *testing.T) {
	tenantID := uuid.New()

	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return model.ExchangeRate{}, fmt.Errorf("not found")
		},
	}

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.87))
	provider := &mockRateProvider{
		fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
			return newRate, nil
		},
	}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	resp, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      tenantID,
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(0.87).Equal(resp.Rate))
	require.Len(t, repo.savedRates, 1)
}

func TestGetExchangeRate_InvalidPair(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	provider := &mockRateProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	_, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      uuid.New(),
		BaseCurrency:  "INVALID",
		QuoteCurrency: "EUR",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid currency pair")
}

func TestGetExchangeRate_ProviderError(t *testing.T) {
	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return model.ExchangeRate{}, fmt.Errorf("not found")
		},
	}
	provider := &mockRateProvider{
		fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
			return valueobject.SpotRate{}, fmt.Errorf("api timeout")
		},
	}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	_, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      uuid.New(),
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch rate from provider")
}

func TestGetExchangeRate_SaveError(t *testing.T) {
	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return model.ExchangeRate{}, fmt.Errorf("not found")
		},
		saveFunc: func(_ context.Context, _ model.ExchangeRate) error {
			return fmt.Errorf("database connection lost")
		},
	}

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.87))
	provider := &mockRateProvider{
		fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
			return newRate, nil
		},
	}
	publisher := &mockEventPublisher{}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	_, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      uuid.New(),
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "save rate")
}

func TestGetExchangeRate_PublishError(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	pair, _ := valueobject.NewCurrencyPair("USD", "EUR")
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.85))

	// Expired rate so it triggers update and event emission.
	expiredRate, _ := model.NewExchangeRate(tenantID, pair, rate, "reuters",
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	repo := &mockExchangeRateRepo{
		findByPairFunc: func(_ context.Context, _ uuid.UUID, _ valueobject.CurrencyPair) (model.ExchangeRate, error) {
			return expiredRate, nil
		},
	}

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.87))
	provider := &mockRateProvider{
		fetchRateFunc: func(_ context.Context, _, _ string) (valueobject.SpotRate, error) {
			return newRate, nil
		},
	}
	publisher := &mockEventPublisher{
		publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
			return fmt.Errorf("broker unreachable")
		},
	}

	uc := usecase.NewGetExchangeRate(repo, provider, publisher)

	_, err := uc.Execute(context.Background(), dto.GetExchangeRateRequest{
		TenantID:      tenantID,
		BaseCurrency:  "USD",
		QuoteCurrency: "EUR",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish events")
}
