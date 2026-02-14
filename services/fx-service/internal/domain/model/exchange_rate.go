package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/fx-service/internal/domain/event"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// ExchangeRate is the root aggregate for the FX bounded context.
// It represents an immutable exchange rate between two currencies at a point in time.
type ExchangeRate struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	pair         valueobject.CurrencyPair
	rate         valueobject.SpotRate
	inverseRate  valueobject.SpotRate
	provider     string
	effectiveAt  time.Time
	expiresAt    time.Time
	version      int
	createdAt    time.Time
	domainEvents []events.DomainEvent
}

// NewExchangeRate creates a new ExchangeRate aggregate with full validation.
func NewExchangeRate(
	tenantID uuid.UUID,
	pair valueobject.CurrencyPair,
	rate valueobject.SpotRate,
	provider string,
	effectiveAt time.Time,
	expiresAt time.Time,
) (ExchangeRate, error) {
	if tenantID == uuid.Nil {
		return ExchangeRate{}, fmt.Errorf("tenant ID is required")
	}
	if rate.IsZero() {
		return ExchangeRate{}, fmt.Errorf("rate is required")
	}
	if provider == "" {
		return ExchangeRate{}, fmt.Errorf("provider is required")
	}
	if effectiveAt.IsZero() {
		return ExchangeRate{}, fmt.Errorf("effective time is required")
	}
	if expiresAt.IsZero() {
		return ExchangeRate{}, fmt.Errorf("expiration time is required")
	}
	if !expiresAt.After(effectiveAt) {
		return ExchangeRate{}, fmt.Errorf("expiration time must be after effective time")
	}

	now := time.Now().UTC()
	return ExchangeRate{
		id:          uuid.New(),
		tenantID:    tenantID,
		pair:        pair,
		rate:        rate,
		inverseRate: rate.Inverse(),
		provider:    provider,
		effectiveAt: effectiveAt,
		expiresAt:   expiresAt,
		version:     1,
		createdAt:   now,
	}, nil
}

// Reconstruct recreates an ExchangeRate from persistence without validation or events.
func Reconstruct(
	id, tenantID uuid.UUID,
	pair valueobject.CurrencyPair,
	rate, inverseRate valueobject.SpotRate,
	provider string,
	effectiveAt, expiresAt time.Time,
	version int,
	createdAt time.Time,
) ExchangeRate {
	return ExchangeRate{
		id:          id,
		tenantID:    tenantID,
		pair:        pair,
		rate:        rate,
		inverseRate: inverseRate,
		provider:    provider,
		effectiveAt: effectiveAt,
		expiresAt:   expiresAt,
		version:     version,
		createdAt:   createdAt,
	}
}

// Update returns a new ExchangeRate with an updated rate and provider.
// This is an immutable operation - the original is unchanged.
// A RateUpdated domain event is emitted on the returned copy.
func (er ExchangeRate) Update(newRate valueobject.SpotRate, provider string, now time.Time) (ExchangeRate, error) {
	if newRate.IsZero() {
		return ExchangeRate{}, fmt.Errorf("new rate must be positive")
	}
	if provider == "" {
		return ExchangeRate{}, fmt.Errorf("provider is required")
	}

	updated := ExchangeRate{
		id:           er.id,
		tenantID:     er.tenantID,
		pair:         er.pair,
		rate:         newRate,
		inverseRate:  newRate.Inverse(),
		provider:     provider,
		effectiveAt:  now,
		expiresAt:    now.Add(er.expiresAt.Sub(er.effectiveAt)), // preserve TTL
		version:      er.version + 1,
		createdAt:    er.createdAt,
		domainEvents: append([]events.DomainEvent{}, er.domainEvents...),
	}

	updated.domainEvents = append(updated.domainEvents,
		event.NewRateUpdated(er.id, er.tenantID, er.pair.String(), newRate.Rate().String(), provider),
	)

	return updated, nil
}

// IsExpired returns true if this rate has expired at the given time.
func (er ExchangeRate) IsExpired(now time.Time) bool {
	return now.After(er.expiresAt)
}

// Convert converts an amount from the base currency to the quote currency using this rate.
func (er ExchangeRate) Convert(amount decimal.Decimal) decimal.Decimal {
	return er.rate.Convert(amount)
}

// Accessors

func (er ExchangeRate) ID() uuid.UUID                      { return er.id }
func (er ExchangeRate) TenantID() uuid.UUID                { return er.tenantID }
func (er ExchangeRate) Pair() valueobject.CurrencyPair     { return er.pair }
func (er ExchangeRate) Rate() valueobject.SpotRate          { return er.rate }
func (er ExchangeRate) InverseRate() valueobject.SpotRate   { return er.inverseRate }
func (er ExchangeRate) Provider() string                    { return er.provider }
func (er ExchangeRate) EffectiveAt() time.Time              { return er.effectiveAt }
func (er ExchangeRate) ExpiresAt() time.Time                { return er.expiresAt }
func (er ExchangeRate) Version() int                        { return er.version }
func (er ExchangeRate) CreatedAt() time.Time                { return er.createdAt }
func (er ExchangeRate) DomainEvents() []events.DomainEvent  { return er.domainEvents }

// ClearDomainEvents returns collected domain events and clears them from the aggregate.
func (er ExchangeRate) ClearDomainEvents() []events.DomainEvent {
	evts := er.domainEvents
	er.domainEvents = nil
	return evts
}
