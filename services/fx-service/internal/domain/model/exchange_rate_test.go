package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

func validPair(t *testing.T) valueobject.CurrencyPair {
	t.Helper()
	pair, err := valueobject.NewCurrencyPair("USD", "EUR")
	require.NoError(t, err)
	return pair
}

func validRate(t *testing.T) valueobject.SpotRate {
	t.Helper()
	rate, err := valueobject.NewSpotRate(decimal.NewFromFloat(0.85))
	require.NoError(t, err)
	return rate
}

func TestNewExchangeRate_Valid(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	tenantID := uuid.New()
	now := time.Now().UTC()
	expires := now.Add(1 * time.Hour)

	er, err := model.NewExchangeRate(tenantID, pair, rate, "reuters", now, expires)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, er.ID())
	assert.Equal(t, tenantID, er.TenantID())
	assert.True(t, pair.Equal(er.Pair()))
	assert.True(t, rate.Equal(er.Rate()))
	assert.False(t, er.InverseRate().IsZero())
	assert.Equal(t, "reuters", er.Provider())
	assert.Equal(t, now, er.EffectiveAt())
	assert.Equal(t, expires, er.ExpiresAt())
	assert.Equal(t, 1, er.Version())
}

func TestNewExchangeRate_NilTenantID(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	now := time.Now().UTC()

	_, err := model.NewExchangeRate(uuid.Nil, pair, rate, "reuters", now, now.Add(time.Hour))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewExchangeRate_EmptyProvider(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	now := time.Now().UTC()

	_, err := model.NewExchangeRate(uuid.New(), pair, rate, "", now, now.Add(time.Hour))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestNewExchangeRate_ExpiresBeforeEffective(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	now := time.Now().UTC()

	_, err := model.NewExchangeRate(uuid.New(), pair, rate, "reuters", now, now.Add(-time.Hour))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiration time must be after effective time")
}

func TestExchangeRate_IsExpired(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	now := time.Now().UTC()
	expires := now.Add(1 * time.Hour)

	er, _ := model.NewExchangeRate(uuid.New(), pair, rate, "reuters", now, expires)

	assert.False(t, er.IsExpired(now))
	assert.False(t, er.IsExpired(now.Add(30*time.Minute)))
	assert.True(t, er.IsExpired(now.Add(2*time.Hour)))
}

func TestExchangeRate_Convert(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t) // 0.85
	now := time.Now().UTC()

	er, _ := model.NewExchangeRate(uuid.New(), pair, rate, "reuters", now, now.Add(time.Hour))

	amount := decimal.NewFromFloat(100.0)
	converted := er.Convert(amount)

	expected := decimal.NewFromFloat(85.0)
	assert.True(t, expected.Equal(converted), "expected %s, got %s", expected, converted)
}

func TestExchangeRate_Update(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	tenantID := uuid.New()
	now := time.Now().UTC()

	original, _ := model.NewExchangeRate(tenantID, pair, rate, "reuters", now, now.Add(time.Hour))

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.90))
	later := now.Add(30 * time.Minute)

	updated, err := original.Update(newRate, "bloomberg", later)

	require.NoError(t, err)

	// Original unchanged.
	assert.True(t, rate.Equal(original.Rate()))
	assert.Equal(t, "reuters", original.Provider())
	assert.Equal(t, 1, original.Version())

	// Updated has new values.
	assert.True(t, newRate.Equal(updated.Rate()))
	assert.Equal(t, "bloomberg", updated.Provider())
	assert.Equal(t, 2, updated.Version())
	assert.Equal(t, later, updated.EffectiveAt())

	// Domain events emitted.
	assert.Len(t, updated.DomainEvents(), 1)
	assert.Equal(t, "fx.rate.updated", updated.DomainEvents()[0].EventType())
}

func TestExchangeRate_Update_EmptyProvider(t *testing.T) {
	pair := validPair(t)
	rate := validRate(t)
	now := time.Now().UTC()

	original, _ := model.NewExchangeRate(uuid.New(), pair, rate, "reuters", now, now.Add(time.Hour))

	newRate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(0.90))

	_, err := original.Update(newRate, "", now)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestExchangeRate_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	pair := validPair(t)
	rate := validRate(t)
	invRate := rate.Inverse()
	now := time.Now().UTC()
	expires := now.Add(time.Hour)

	er := model.Reconstruct(id, tenantID, pair, rate, invRate, "reuters", now, expires, 3, now)

	assert.Equal(t, id, er.ID())
	assert.Equal(t, tenantID, er.TenantID())
	assert.Equal(t, 3, er.Version())
	assert.Equal(t, "reuters", er.Provider())
	assert.Empty(t, er.DomainEvents())
}

func TestExchangeRate_InverseRate(t *testing.T) {
	pair := validPair(t)
	rate, _ := valueobject.NewSpotRate(decimal.NewFromFloat(2.0))
	now := time.Now().UTC()

	er, _ := model.NewExchangeRate(uuid.New(), pair, rate, "reuters", now, now.Add(time.Hour))

	assert.True(t, decimal.NewFromFloat(0.5).Equal(er.InverseRate().Rate()))
}
