package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// ExchangeRateRepository defines persistence operations for exchange rates.
type ExchangeRateRepository interface {
	// Save persists an exchange rate (insert or update).
	Save(ctx context.Context, rate model.ExchangeRate) error

	// FindByPair retrieves the latest exchange rate for a currency pair within a tenant.
	FindByPair(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error)

	// FindLatest retrieves the most recent exchange rate for a pair across all tenants.
	FindLatest(ctx context.Context, pair valueobject.CurrencyPair) (model.ExchangeRate, error)

	// ListByBase returns all exchange rates with the given base currency for a tenant.
	ListByBase(ctx context.Context, tenantID uuid.UUID, baseCurrency string, asOf time.Time) ([]model.ExchangeRate, error)
}

// RateProvider is a port for external exchange rate data sources.
type RateProvider interface {
	// FetchRate fetches the current spot rate from an external provider.
	FetchRate(ctx context.Context, base, quote string) (valueobject.SpotRate, error)
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...events.DomainEvent) error
}
