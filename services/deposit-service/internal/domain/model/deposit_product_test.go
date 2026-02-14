package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

func newTestTiers(t *testing.T) []valueobject.InterestTier {
	t.Helper()
	tier1, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(9999), 100)
	require.NoError(t, err)
	tier2, err := valueobject.NewInterestTier(decimal.NewFromInt(10000), decimal.NewFromInt(99999), 250)
	require.NoError(t, err)
	tier3, err := valueobject.NewInterestTier(decimal.NewFromInt(100000), decimal.NewFromInt(999999999), 350)
	require.NoError(t, err)
	return []valueobject.InterestTier{tier1, tier2, tier3}
}

func TestNewDepositProduct_Valid(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Savings Plus", "USD", tiers, 0)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, product.ID())
	assert.Equal(t, tenantID, product.TenantID())
	assert.Equal(t, "Savings Plus", product.Name())
	assert.Equal(t, "USD", product.Currency())
	assert.Len(t, product.Tiers(), 3)
	assert.Equal(t, 0, product.TermDays())
	assert.True(t, product.IsActive())
	assert.Equal(t, 1, product.Version())
	assert.False(t, product.CreatedAt().IsZero())
	assert.False(t, product.UpdatedAt().IsZero())
	assert.False(t, product.IsTermDeposit())
}

func TestNewDepositProduct_TermDeposit(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Fixed 90-Day", "EUR", tiers, 90)
	require.NoError(t, err)

	assert.Equal(t, 90, product.TermDays())
	assert.True(t, product.IsTermDeposit())
}

func TestNewDepositProduct_MissingTenantID(t *testing.T) {
	tiers := newTestTiers(t)
	_, err := model.NewDepositProduct(uuid.Nil, "Test", "USD", tiers, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewDepositProduct_MissingName(t *testing.T) {
	tiers := newTestTiers(t)
	_, err := model.NewDepositProduct(uuid.New(), "", "USD", tiers, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product name is required")
}

func TestNewDepositProduct_MissingCurrency(t *testing.T) {
	tiers := newTestTiers(t)
	_, err := model.NewDepositProduct(uuid.New(), "Test", "", tiers, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency is required")
}

func TestNewDepositProduct_InvalidCurrency(t *testing.T) {
	tiers := newTestTiers(t)
	_, err := model.NewDepositProduct(uuid.New(), "Test", "US", tiers, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency must be a 3-letter ISO code")
}

func TestNewDepositProduct_EmptyTiers(t *testing.T) {
	_, err := model.NewDepositProduct(uuid.New(), "Test", "USD", nil, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one interest tier is required")

	_, err = model.NewDepositProduct(uuid.New(), "Test", "USD", []valueobject.InterestTier{}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one interest tier is required")
}

func TestNewDepositProduct_NegativeTermDays(t *testing.T) {
	tiers := newTestTiers(t)
	_, err := model.NewDepositProduct(uuid.New(), "Test", "USD", tiers, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "term days must not be negative")
}

func TestNewDepositProduct_OverlappingTiers(t *testing.T) {
	tier1, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(10000), 100)
	require.NoError(t, err)
	tier2, err := valueobject.NewInterestTier(decimal.NewFromInt(5000), decimal.NewFromInt(50000), 200)
	require.NoError(t, err)

	_, err = model.NewDepositProduct(uuid.New(), "Test", "USD", []valueobject.InterestTier{tier1, tier2}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interest tiers overlap")
}

func TestNewDepositProduct_AdjacentTiers_NoOverlap(t *testing.T) {
	tier1, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(9999), 100)
	require.NoError(t, err)
	tier2, err := valueobject.NewInterestTier(decimal.NewFromInt(10000), decimal.NewFromInt(50000), 200)
	require.NoError(t, err)

	product, err := model.NewDepositProduct(uuid.New(), "Test", "USD", []valueobject.InterestTier{tier1, tier2}, 0)
	require.NoError(t, err)
	assert.Len(t, product.Tiers(), 2)
}

func TestDepositProduct_FindApplicableTier(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Test", "USD", tiers, 0)
	require.NoError(t, err)

	// Low balance -> tier 1 (0-9999, 100 bps)
	tier, err := product.FindApplicableTier(decimal.NewFromInt(5000))
	require.NoError(t, err)
	assert.Equal(t, 100, tier.RateBps())

	// Medium balance -> tier 2 (10000-99999, 250 bps)
	tier, err = product.FindApplicableTier(decimal.NewFromInt(25000))
	require.NoError(t, err)
	assert.Equal(t, 250, tier.RateBps())

	// High balance -> tier 3 (100000+, 350 bps)
	tier, err = product.FindApplicableTier(decimal.NewFromInt(500000))
	require.NoError(t, err)
	assert.Equal(t, 350, tier.RateBps())
}

func TestDepositProduct_FindApplicableTier_NoMatch(t *testing.T) {
	tier, err := valueobject.NewInterestTier(decimal.NewFromInt(1000), decimal.NewFromInt(50000), 250)
	require.NoError(t, err)

	product, err := model.NewDepositProduct(uuid.New(), "Test", "USD", []valueobject.InterestTier{tier}, 0)
	require.NoError(t, err)

	_, err = product.FindApplicableTier(decimal.NewFromInt(500))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no applicable tier")
}

func TestDepositProduct_FindApplicableTier_AtBoundary(t *testing.T) {
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(uuid.New(), "Test", "USD", tiers, 0)
	require.NoError(t, err)

	// Exactly at min boundary of tier 2
	tier, err := product.FindApplicableTier(decimal.NewFromInt(10000))
	require.NoError(t, err)
	assert.Equal(t, 250, tier.RateBps())

	// Exactly at max boundary of tier 1
	tier, err = product.FindApplicableTier(decimal.NewFromInt(9999))
	require.NoError(t, err)
	assert.Equal(t, 100, tier.RateBps())
}

func TestDepositProduct_UpdateTiers(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Test", "USD", tiers, 0)
	require.NoError(t, err)

	newTier, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(999999), 500)
	require.NoError(t, err)

	now := time.Now().UTC()
	updated, err := product.UpdateTiers([]valueobject.InterestTier{newTier}, now)
	require.NoError(t, err)

	assert.Len(t, updated.Tiers(), 1)
	assert.Equal(t, 500, updated.Tiers()[0].RateBps())
	assert.Equal(t, 2, updated.Version())
	assert.Equal(t, now, updated.UpdatedAt())

	// Original unchanged
	assert.Len(t, product.Tiers(), 3)
	assert.Equal(t, 1, product.Version())
}

func TestDepositProduct_UpdateTiers_InactiveProduct(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Test", "USD", tiers, 0)
	require.NoError(t, err)

	now := time.Now().UTC()
	deactivated, err := product.Deactivate(now)
	require.NoError(t, err)

	newTier, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(999999), 500)
	require.NoError(t, err)

	_, err = deactivated.UpdateTiers([]valueobject.InterestTier{newTier}, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update tiers on an inactive product")
}

func TestDepositProduct_Deactivate(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Test", "USD", tiers, 0)
	require.NoError(t, err)
	assert.True(t, product.IsActive())

	now := time.Now().UTC()
	deactivated, err := product.Deactivate(now)
	require.NoError(t, err)

	assert.False(t, deactivated.IsActive())
	assert.Equal(t, 2, deactivated.Version())
	assert.Equal(t, now, deactivated.UpdatedAt())

	// Original unchanged
	assert.True(t, product.IsActive())
	assert.Equal(t, 1, product.Version())
}

func TestDepositProduct_Deactivate_AlreadyInactive(t *testing.T) {
	tenantID := uuid.New()
	tiers := newTestTiers(t)

	product, err := model.NewDepositProduct(tenantID, "Test", "USD", tiers, 0)
	require.NoError(t, err)

	now := time.Now().UTC()
	deactivated, err := product.Deactivate(now)
	require.NoError(t, err)

	_, err = deactivated.Deactivate(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product is already inactive")
}

func TestDepositProduct_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	tiers := newTestTiers(t)
	createdAt := time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.June, 2, 0, 0, 0, 0, time.UTC)

	product := model.ReconstructProduct(
		id, tenantID, "Reconstructed", "EUR", tiers, 180, true, 3, createdAt, updatedAt,
	)

	assert.Equal(t, id, product.ID())
	assert.Equal(t, tenantID, product.TenantID())
	assert.Equal(t, "Reconstructed", product.Name())
	assert.Equal(t, "EUR", product.Currency())
	assert.Len(t, product.Tiers(), 3)
	assert.Equal(t, 180, product.TermDays())
	assert.True(t, product.IsActive())
	assert.Equal(t, 3, product.Version())
	assert.Equal(t, createdAt, product.CreatedAt())
	assert.Equal(t, updatedAt, product.UpdatedAt())
	assert.True(t, product.IsTermDeposit())
}
