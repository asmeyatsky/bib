package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/card-service/internal/domain/event"
	"github.com/bibbank/bib/services/card-service/internal/domain/model"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

func TestCardFullLifecycle(t *testing.T) {
	tenantID := uuid.New()
	accountID := uuid.New()
	dailyLimit := decimal.NewFromInt(1000)
	monthlyLimit := decimal.NewFromInt(5000)
	now := time.Now().UTC()

	// --- Step 1: Issue card ---
	card, err := model.NewCard(tenantID, accountID, valueobject.CardTypeVirtual, "USD", dailyLimit, monthlyLimit)
	require.NoError(t, err)

	assert.Equal(t, tenantID, card.TenantID())
	assert.Equal(t, accountID, card.AccountID())
	assert.Equal(t, valueobject.CardTypeVirtual, card.CardType())
	assert.Equal(t, valueobject.CardStatusPending, card.Status())
	assert.Equal(t, "USD", card.Currency())
	assert.True(t, card.DailyLimit().Equal(dailyLimit))
	assert.True(t, card.MonthlyLimit().Equal(monthlyLimit))
	assert.True(t, card.DailySpent().IsZero())
	assert.True(t, card.MonthlySpent().IsZero())
	assert.Equal(t, 1, card.Version())

	// Verify CardIssued event was emitted.
	events := card.DomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(event.CardIssued)
	assert.True(t, ok, "expected CardIssued event")

	card = card.ClearEvents()

	// --- Step 2: Activate card ---
	card, err = card.Activate(now)
	require.NoError(t, err)

	assert.Equal(t, valueobject.CardStatusActive, card.Status())
	assert.Equal(t, 2, card.Version())

	events = card.DomainEvents()
	require.Len(t, events, 1)
	_, ok = events[0].(event.CardActivated)
	assert.True(t, ok, "expected CardActivated event")

	card = card.ClearEvents()

	// --- Step 3: Authorize transaction within limits ---
	amount := decimal.NewFromInt(500)
	card, authCode, err := card.AuthorizeTransaction(amount, "Coffee Shop", "5814", now)
	require.NoError(t, err)

	assert.NotEmpty(t, authCode)
	assert.Len(t, authCode, 8)
	assert.True(t, card.DailySpent().Equal(amount))
	assert.True(t, card.MonthlySpent().Equal(amount))
	assert.Equal(t, 3, card.Version())

	events = card.DomainEvents()
	require.Len(t, events, 1)
	txnEvent, ok := events[0].(event.TransactionAuthorized)
	assert.True(t, ok, "expected TransactionAuthorized event")
	assert.True(t, txnEvent.Amount.Equal(amount))
	assert.Equal(t, "Coffee Shop", txnEvent.MerchantName)
	assert.Equal(t, authCode, txnEvent.AuthCode)

	card = card.ClearEvents()

	// --- Step 4: Authorize transaction that exceeds daily limit ---
	overLimitAmount := decimal.NewFromInt(600) // 500 + 600 = 1100 > 1000 daily limit
	card, _, err = card.AuthorizeTransaction(overLimitAmount, "Electronics Store", "5732", now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily spending limit exceeded")

	// Daily spent should NOT have increased.
	assert.True(t, card.DailySpent().Equal(amount), "daily spent should remain at 500")

	// Verify TransactionDeclined event was emitted.
	events = card.DomainEvents()
	require.Len(t, events, 1)
	_, ok = events[0].(event.TransactionDeclined)
	assert.True(t, ok, "expected TransactionDeclined event")

	card = card.ClearEvents()

	// --- Step 5: Freeze card ---
	card, err = card.Freeze(now)
	require.NoError(t, err)

	assert.Equal(t, valueobject.CardStatusFrozen, card.Status())

	events = card.DomainEvents()
	require.Len(t, events, 1)
	_, ok = events[0].(event.CardFrozen)
	assert.True(t, ok, "expected CardFrozen event")

	card = card.ClearEvents()

	// --- Step 6: Authorize transaction on frozen card -> error ---
	card, _, err = card.AuthorizeTransaction(decimal.NewFromInt(10), "Grocery", "5411", now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "card is not usable")

	events = card.DomainEvents()
	require.Len(t, events, 1)
	_, ok = events[0].(event.TransactionDeclined)
	assert.True(t, ok, "expected TransactionDeclined event for frozen card")
}

func TestCard_NewCard_Validation(t *testing.T) {
	tenantID := uuid.New()
	accountID := uuid.New()

	tests := []struct {
		name         string
		cardType     valueobject.CardType
		currency     string
		dailyLimit   decimal.Decimal
		monthlyLimit decimal.Decimal
		wantErr      string
		tenantID     uuid.UUID
		accountID    uuid.UUID
	}{
		{
			name:         "nil tenant ID",
			tenantID:     uuid.Nil,
			accountID:    accountID,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "USD",
			dailyLimit:   decimal.NewFromInt(1000),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "tenant ID is required",
		},
		{
			name:         "nil account ID",
			tenantID:     tenantID,
			accountID:    uuid.Nil,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "USD",
			dailyLimit:   decimal.NewFromInt(1000),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "account ID is required",
		},
		{
			name:         "empty currency",
			tenantID:     tenantID,
			accountID:    accountID,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "",
			dailyLimit:   decimal.NewFromInt(1000),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "currency is required",
		},
		{
			name:         "invalid currency length",
			tenantID:     tenantID,
			accountID:    accountID,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "US",
			dailyLimit:   decimal.NewFromInt(1000),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "currency must be a 3-letter ISO code",
		},
		{
			name:         "negative daily limit",
			tenantID:     tenantID,
			accountID:    accountID,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "USD",
			dailyLimit:   decimal.NewFromInt(-100),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "daily limit must be positive",
		},
		{
			name:         "daily exceeds monthly",
			tenantID:     tenantID,
			accountID:    accountID,
			cardType:     valueobject.CardTypeVirtual,
			currency:     "USD",
			dailyLimit:   decimal.NewFromInt(6000),
			monthlyLimit: decimal.NewFromInt(5000),
			wantErr:      "daily limit cannot exceed monthly limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.NewCard(tt.tenantID, tt.accountID, tt.cardType, tt.currency, tt.dailyLimit, tt.monthlyLimit)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCard_StatusTransitions(t *testing.T) {
	now := time.Now().UTC()
	card := createActiveCard(t)

	// Cannot activate an already active card.
	_, err := card.Activate(now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be PENDING")

	// Freeze the card.
	frozen, err := card.Freeze(now)
	require.NoError(t, err)

	// Cannot freeze a frozen card.
	_, err = frozen.Freeze(now)
	require.Error(t, err)

	// Unfreeze the card.
	unfrozen, err := frozen.Unfreeze(now)
	require.NoError(t, err)
	assert.Equal(t, valueobject.CardStatusActive, unfrozen.Status())

	// Cancel from any state.
	canceled, err := unfrozen.Cancel(now)
	require.NoError(t, err)
	assert.Equal(t, valueobject.CardStatusCanceled, canceled.Status())

	// Cannot cancel an already canceled card.
	_, err = canceled.Cancel(now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already canceled")
}

func TestCard_ResetSpending(t *testing.T) {
	now := time.Now().UTC()
	card := createActiveCard(t)

	// Authorize a transaction.
	card, _, err := card.AuthorizeTransaction(decimal.NewFromInt(100), "Test", "0000", now)
	require.NoError(t, err)
	assert.True(t, card.DailySpent().Equal(decimal.NewFromInt(100)))
	assert.True(t, card.MonthlySpent().Equal(decimal.NewFromInt(100)))

	// Reset daily spend.
	card = card.ResetDailySpend(now)
	assert.True(t, card.DailySpent().IsZero())
	assert.True(t, card.MonthlySpent().Equal(decimal.NewFromInt(100))) // Monthly unchanged.

	// Reset monthly spend.
	card = card.ResetMonthlySpend(now)
	assert.True(t, card.MonthlySpent().IsZero())
}

func TestCardNumber_Expiry(t *testing.T) {
	cn, err := valueobject.NewCardNumber("1234", "12", "2025")
	require.NoError(t, err)

	// Before expiry.
	beforeExpiry := time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)
	assert.False(t, cn.IsExpired(beforeExpiry))

	// After expiry.
	afterExpiry := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.True(t, cn.IsExpired(afterExpiry))
}

func TestCardNumber_Validation(t *testing.T) {
	// Valid.
	_, err := valueobject.NewCardNumber("1234", "06", "2027")
	require.NoError(t, err)

	// Invalid last four.
	_, err = valueobject.NewCardNumber("123", "06", "2027")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last four must be exactly 4 digits")

	// Invalid last four (letters).
	_, err = valueobject.NewCardNumber("12ab", "06", "2027")
	require.Error(t, err)

	// Invalid month.
	_, err = valueobject.NewCardNumber("1234", "13", "2027")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry month must be 01-12")

	// Invalid year.
	_, err = valueobject.NewCardNumber("1234", "06", "27")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry year must be exactly 4 digits")
}

// createActiveCard is a test helper that creates a card and activates it.
func createActiveCard(t *testing.T) model.Card {
	t.Helper()

	card, err := model.NewCard(
		uuid.New(),
		uuid.New(),
		valueobject.CardTypeVirtual,
		"USD",
		decimal.NewFromInt(1000),
		decimal.NewFromInt(5000),
	)
	require.NoError(t, err)

	card = card.ClearEvents()
	card, err = card.Activate(time.Now().UTC())
	require.NoError(t, err)

	return card.ClearEvents()
}
