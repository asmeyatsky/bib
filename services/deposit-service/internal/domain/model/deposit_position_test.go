package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
)

func TestNewDepositPosition_Valid(t *testing.T) {
	tenantID := uuid.New()
	accountID := uuid.New()
	productID := uuid.New()
	principal := decimal.NewFromInt(10000)

	pos, err := model.NewDepositPosition(tenantID, accountID, productID, principal, "USD", nil)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, pos.ID())
	assert.Equal(t, tenantID, pos.TenantID())
	assert.Equal(t, accountID, pos.AccountID())
	assert.Equal(t, productID, pos.ProductID())
	assert.True(t, pos.Principal().Equal(principal))
	assert.Equal(t, "USD", pos.Currency())
	assert.True(t, pos.AccruedInterest().IsZero())
	assert.Equal(t, model.PositionStatusActive, pos.Status())
	assert.False(t, pos.OpenedAt().IsZero())
	assert.Nil(t, pos.MaturityDate())
	assert.Equal(t, 1, pos.Version())
	assert.Len(t, pos.DomainEvents(), 1)
	assert.Equal(t, "deposit.position.opened", pos.DomainEvents()[0].EventType())
}

func TestNewDepositPosition_WithMaturityDate(t *testing.T) {
	maturity := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)
	pos, err := model.NewDepositPosition(
		uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(5000), "EUR", &maturity,
	)
	require.NoError(t, err)

	require.NotNil(t, pos.MaturityDate())
	assert.Equal(t, maturity, *pos.MaturityDate())
}

func TestNewDepositPosition_MissingTenantID(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.Nil, uuid.New(), uuid.New(), decimal.NewFromInt(100), "USD", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewDepositPosition_MissingAccountID(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.New(), uuid.Nil, uuid.New(), decimal.NewFromInt(100), "USD", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID is required")
}

func TestNewDepositPosition_MissingProductID(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.New(), uuid.New(), uuid.Nil, decimal.NewFromInt(100), "USD", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID is required")
}

func TestNewDepositPosition_ZeroPrincipal(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.New(), uuid.New(), uuid.New(), decimal.Zero, "USD", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "principal must be positive")
}

func TestNewDepositPosition_NegativePrincipal(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(-100), "USD", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "principal must be positive")
}

func TestNewDepositPosition_InvalidCurrency(t *testing.T) {
	_, err := model.NewDepositPosition(uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(100), "US", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency must be a 3-letter ISO code")
}

func TestDepositPosition_AccrueInterest_30Days(t *testing.T) {
	// $10,000 at 250 bps for 30 days
	// Expected: $10,000 * 0.025 / 365 * 30 = $20.5479...
	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		principal, "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	// Daily rate for 250 bps: 0.025 / 365
	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))

	asOf := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC) // 30 days later
	accrued, err := pos.AccrueInterest(dailyRate, asOf)
	require.NoError(t, err)

	// Expected interest: 10000 * 0.025/365 * 30 = 10000 * 0.00006849315... * 30
	expectedInterest := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)

	assert.True(t, accrued.AccruedInterest().Equal(expectedInterest),
		"expected %s, got %s", expectedInterest, accrued.AccruedInterest())
	assert.Equal(t, asOf, accrued.LastAccrualDate())
	assert.Equal(t, 2, accrued.Version())

	// Verify the accrued amount is approximately $20.55
	assert.True(t, accrued.AccruedInterest().GreaterThan(decimal.NewFromFloat(20.0)),
		"accrued interest %s should be > $20", accrued.AccruedInterest())
	assert.True(t, accrued.AccruedInterest().LessThan(decimal.NewFromFloat(21.0)),
		"accrued interest %s should be < $21", accrued.AccruedInterest())

	// Domain event emitted
	events := accrued.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "deposit.interest.accrued", events[0].EventType())
}

func TestDepositPosition_AccrueInterest_SameDay(t *testing.T) {
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))

	// Accrue on the same day - no interest should accrue
	accrued, err := pos.AccrueInterest(dailyRate, lastAccrual)
	require.NoError(t, err)
	assert.True(t, accrued.AccruedInterest().IsZero())
}

func TestDepositPosition_AccrueInterest_NotActive(t *testing.T) {
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusClosed,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	asOf := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	_, err := pos.AccrueInterest(dailyRate, asOf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only accrue interest on ACTIVE positions")
}

func TestDepositPosition_AccrueInterest_PastDate(t *testing.T) {
	lastAccrual := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	asOf := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC) // before last accrual

	_, err := pos.AccrueInterest(dailyRate, asOf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accrual date")
	assert.Contains(t, err.Error(), "before last accrual date")
}

func TestDepositPosition_AccrueInterest_Cumulative(t *testing.T) {
	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		principal, "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))

	// Accrue 10 days
	asOf1 := time.Date(2024, time.January, 11, 0, 0, 0, 0, time.UTC)
	accrued1, err := pos.AccrueInterest(dailyRate, asOf1)
	require.NoError(t, err)
	interest1 := accrued1.AccruedInterest()
	assert.False(t, interest1.IsZero())

	// Accrue another 20 days
	asOf2 := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC)
	accrued2, err := accrued1.AccrueInterest(dailyRate, asOf2)
	require.NoError(t, err)
	interest2 := accrued2.AccruedInterest()

	// Total should be 30 days of interest
	expectedTotal := principal.Mul(dailyRate).Mul(decimal.NewFromInt(30)).Round(4)
	assert.True(t, interest2.Equal(expectedTotal),
		"expected %s, got %s", expectedTotal, interest2)
}

func TestDepositPosition_AccrueInterest_Immutable(t *testing.T) {
	principal := decimal.NewFromInt(10000)
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		principal, "USD", decimal.Zero, model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	dailyRate := decimal.NewFromFloat(0.025).Div(decimal.NewFromInt(365))
	asOf := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	_, err := pos.AccrueInterest(dailyRate, asOf)
	require.NoError(t, err)

	// Original should remain unchanged
	assert.True(t, pos.AccruedInterest().IsZero())
	assert.Equal(t, lastAccrual, pos.LastAccrualDate())
	assert.Equal(t, 1, pos.Version())
}

func TestDepositPosition_Mature(t *testing.T) {
	pos, err := model.NewDepositPosition(
		uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", nil,
	)
	require.NoError(t, err)

	now := time.Now().UTC()
	matured, err := pos.Mature(now)
	require.NoError(t, err)

	assert.Equal(t, model.PositionStatusMatured, matured.Status())
	assert.Equal(t, 2, matured.Version())

	// Check domain events: DepositOpened + DepositMatured
	events := matured.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, "deposit.position.opened", events[0].EventType())
	assert.Equal(t, "deposit.position.matured", events[1].EventType())

	// Original unchanged
	assert.Equal(t, model.PositionStatusActive, pos.Status())
}

func TestDepositPosition_Mature_NotActive(t *testing.T) {
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusClosed,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	_, err := pos.Mature(time.Now().UTC())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only mature ACTIVE positions")
}

func TestDepositPosition_Close_FromActive(t *testing.T) {
	pos, err := model.NewDepositPosition(
		uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", nil,
	)
	require.NoError(t, err)

	now := time.Now().UTC()
	closed, err := pos.Close(now)
	require.NoError(t, err)

	assert.Equal(t, model.PositionStatusClosed, closed.Status())
	assert.Equal(t, 2, closed.Version())

	events := closed.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, "deposit.position.opened", events[0].EventType())
	assert.Equal(t, "deposit.position.closed", events[1].EventType())
}

func TestDepositPosition_Close_FromMatured(t *testing.T) {
	pos, err := model.NewDepositPosition(
		uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", nil,
	)
	require.NoError(t, err)

	now := time.Now().UTC()
	matured, err := pos.Mature(now)
	require.NoError(t, err)

	closed, err := matured.Close(now)
	require.NoError(t, err)

	assert.Equal(t, model.PositionStatusClosed, closed.Status())
	assert.Equal(t, 3, closed.Version())
}

func TestDepositPosition_Close_FromClosed(t *testing.T) {
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", decimal.Zero, model.PositionStatusClosed,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	_, err := pos.Close(time.Now().UTC())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only close ACTIVE or MATURED positions")
}

func TestDepositPosition_TotalBalance(t *testing.T) {
	lastAccrual := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromFloat(10000.00), "USD", decimal.NewFromFloat(123.45), model.PositionStatusActive,
		lastAccrual, nil, lastAccrual, 1,
		lastAccrual, lastAccrual,
	)

	expected := decimal.NewFromFloat(10123.45)
	assert.True(t, pos.TotalBalance().Equal(expected),
		"expected %s, got %s", expected, pos.TotalBalance())
}

func TestDepositPosition_ClearDomainEvents(t *testing.T) {
	pos, err := model.NewDepositPosition(
		uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(10000), "USD", nil,
	)
	require.NoError(t, err)
	require.Len(t, pos.DomainEvents(), 1)

	cleared := pos.ClearDomainEvents()
	assert.Len(t, cleared, 1)
	assert.Equal(t, "deposit.position.opened", cleared[0].EventType())
}

func TestDepositPosition_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	accountID := uuid.New()
	productID := uuid.New()
	principal := decimal.NewFromFloat(50000.00)
	accrued := decimal.NewFromFloat(250.75)
	openedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	maturity := time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)
	lastAccrual := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)

	pos := model.ReconstructPosition(
		id, tenantID, accountID, productID,
		principal, "EUR", accrued, model.PositionStatusActive,
		openedAt, &maturity, lastAccrual, 5, createdAt, updatedAt,
	)

	assert.Equal(t, id, pos.ID())
	assert.Equal(t, tenantID, pos.TenantID())
	assert.Equal(t, accountID, pos.AccountID())
	assert.Equal(t, productID, pos.ProductID())
	assert.True(t, pos.Principal().Equal(principal))
	assert.Equal(t, "EUR", pos.Currency())
	assert.True(t, pos.AccruedInterest().Equal(accrued))
	assert.Equal(t, model.PositionStatusActive, pos.Status())
	assert.Equal(t, openedAt, pos.OpenedAt())
	require.NotNil(t, pos.MaturityDate())
	assert.Equal(t, maturity, *pos.MaturityDate())
	assert.Equal(t, lastAccrual, pos.LastAccrualDate())
	assert.Equal(t, 5, pos.Version())
	assert.Equal(t, createdAt, pos.CreatedAt())
	assert.Equal(t, updatedAt, pos.UpdatedAt())
	assert.Empty(t, pos.DomainEvents()) // reconstruct does not generate events
}

func TestPositionStatus_Constants(t *testing.T) {
	assert.Equal(t, model.PositionStatus("ACTIVE"), model.PositionStatusActive)
	assert.Equal(t, model.PositionStatus("MATURED"), model.PositionStatusMatured)
	assert.Equal(t, model.PositionStatus("CLOSED"), model.PositionStatusClosed)
}
