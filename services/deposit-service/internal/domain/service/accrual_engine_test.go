package service_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/service"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

func newTestProduct(t *testing.T) model.DepositProduct {
	t.Helper()
	tier1, err := valueobject.NewInterestTier(decimal.NewFromInt(0), decimal.NewFromInt(9999), 100)
	require.NoError(t, err)
	tier2, err := valueobject.NewInterestTier(decimal.NewFromInt(10000), decimal.NewFromInt(99999), 250)
	require.NoError(t, err)
	tier3, err := valueobject.NewInterestTier(decimal.NewFromInt(100000), decimal.NewFromInt(999999999), 350)
	require.NoError(t, err)

	product, err := model.NewDepositProduct(
		uuid.New(), "Test Savings", "USD",
		[]valueobject.InterestTier{tier1, tier2, tier3}, 0,
	)
	require.NoError(t, err)
	return product
}

func newTestPosition(t *testing.T, productID uuid.UUID, principal decimal.Decimal, lastAccrual time.Time) model.DepositPosition {
	t.Helper()
	return model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), productID,
		principal, "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)
}

func TestAccrualEngine_AccrueForPosition_LowBalance(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	// $5,000 balance -> tier 1 (100 bps = 1%)
	principal := decimal.NewFromInt(5000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC) // 30 days
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// Expected: 5000 * (0.01 / 365) * 30
	dailyRate := decimal.NewFromFloat(0.01).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_MediumBalance(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	// $25,000 balance -> tier 2 (250 bps = 2.5%)
	principal := decimal.NewFromInt(25000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC) // 30 days
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// Expected: 25000 * (0.025 / 365) * 30
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_HighBalance(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	// $500,000 balance -> tier 3 (350 bps = 3.5%)
	principal := decimal.NewFromInt(500000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC) // 30 days
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// Expected: 500000 * (0.035 / 365) * 30
	dailyRate := decimal.NewFromFloat(0.035).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_TenThousandAt250Bps(t *testing.T) {
	// Classic test: $10,000 at 250 bps for 30 days
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC) // 30 days
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// $10,000 falls in tier 2 (10000-99999, 250 bps)
	// Expected: 10000 * 0.025 / 365 * 30 = 20.5479...
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())

	// Verify approximately $20.55
	assert.True(t, accrued.AccruedInterest().GreaterThan(decimal.NewFromFloat(20.0)))
	assert.True(t, accrued.AccruedInterest().LessThan(decimal.NewFromFloat(21.0)))
}

func TestAccrualEngine_AccrueForPosition_WithExistingAccrual(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	// Position with $25,000 principal and $50 already accrued (total $25,050)
	// Still falls in tier 2 (10000-99999)
	principal := decimal.NewFromInt(25000)
	existingAccrual := decimal.NewFromFloat(50.00)
	lastAccrual := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	position := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), product.ID(),
		principal, "USD", existingAccrual, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 2,
		lastAccrual, lastAccrual,
	)

	asOf := time.Date(2024, time.March, 2, 0, 0, 0, 0, time.UTC) // 30 days
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// New interest based on principal (not total balance - interest is on principal)
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	newInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)
	expectedTotal := existingAccrual.Add(newInterest)

	assert.True(t, accrued.AccruedInterest().Equal(expectedTotal),
		"expected %s, got %s", expectedTotal, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_InactivePosition(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), product.ID(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusClosed,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	asOf := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	_, err := engine.AccrueForPosition(position, product, asOf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestAccrualEngine_AccrueForPosition_NoApplicableTier(t *testing.T) {
	engine := service.NewAccrualEngine()

	// Product with tier starting at $1000
	tier, err := valueobject.NewInterestTier(decimal.NewFromInt(1000), decimal.NewFromInt(100000), 250)
	require.NoError(t, err)
	product, err := model.NewDepositProduct(uuid.New(), "Test", "USD", []valueobject.InterestTier{tier}, 0)
	require.NoError(t, err)

	// Position with $500 (below tier minimum)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), decimal.NewFromInt(500), lastAccrual)

	asOf := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	_, err = engine.AccrueForPosition(position, product, asOf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find tier")
}

func TestAccrualEngine_AccrueForPosition_OneDay(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC) // 1 day
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// Expected: 10000 * 0.025/365 * 1
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_FullYear(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC) // 366 days (2024 is leap year)
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	// For 366 days (leap year 2024): 10000 * 0.025/365 * 366
	// This is slightly more than the annual rate due to the extra day
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(366)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
}

func TestAccrualEngine_AccrueForPosition_DomainEvents(t *testing.T) {
	engine := service.NewAccrualEngine()
	product := newTestProduct(t)

	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	position := newTestPosition(t, product.ID(), principal, lastAccrual)

	asOf := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	accrued, err := engine.AccrueForPosition(position, product, asOf)
	require.NoError(t, err)

	events := accrued.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "deposit.interest.accrued", events[0].EventType())
	assert.Equal(t, position.ID(), events[0].AggregateID())
}
