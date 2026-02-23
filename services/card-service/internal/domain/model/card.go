package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/card-service/internal/domain/event"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// Card is the aggregate root for card management.
// It encapsulates all card state and enforces business invariants.
type Card struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	accountID    uuid.UUID
	cardType     valueobject.CardType
	status       valueobject.CardStatus
	cardNumber   valueobject.CardNumber
	currency     string
	dailyLimit   decimal.Decimal
	monthlyLimit decimal.Decimal
	dailySpent   decimal.Decimal
	monthlySpent decimal.Decimal
	version      int
	createdAt    time.Time
	updatedAt    time.Time
	domainEvents []events.DomainEvent
}

// NewCard creates a new Card aggregate in PENDING status.
// A random last-four and expiry (3 years out) are generated for the card number.
func NewCard(
	tenantID, accountID uuid.UUID,
	cardType valueobject.CardType,
	currency string,
	dailyLimit, monthlyLimit decimal.Decimal,
) (Card, error) {
	if tenantID == uuid.Nil {
		return Card{}, fmt.Errorf("tenant ID is required")
	}
	if accountID == uuid.Nil {
		return Card{}, fmt.Errorf("account ID is required")
	}
	if currency == "" {
		return Card{}, fmt.Errorf("currency is required")
	}
	if len(currency) != 3 {
		return Card{}, fmt.Errorf("currency must be a 3-letter ISO code")
	}
	if dailyLimit.IsNegative() || dailyLimit.IsZero() {
		return Card{}, fmt.Errorf("daily limit must be positive")
	}
	if monthlyLimit.IsNegative() || monthlyLimit.IsZero() {
		return Card{}, fmt.Errorf("monthly limit must be positive")
	}
	if dailyLimit.GreaterThan(monthlyLimit) {
		return Card{}, fmt.Errorf("daily limit cannot exceed monthly limit")
	}

	lastFour := generateRandomLastFour()
	now := time.Now().UTC()
	expiry := now.AddDate(3, 0, 0) // 3-year expiry
	expiryMonth := fmt.Sprintf("%02d", int(expiry.Month()))
	expiryYear := fmt.Sprintf("%d", expiry.Year())

	cardNumber, err := valueobject.NewCardNumber(lastFour, expiryMonth, expiryYear)
	if err != nil {
		return Card{}, fmt.Errorf("failed to create card number: %w", err)
	}

	id := uuid.New()

	c := Card{
		id:           id,
		tenantID:     tenantID,
		accountID:    accountID,
		cardType:     cardType,
		status:       valueobject.CardStatusPending,
		cardNumber:   cardNumber,
		currency:     currency,
		dailyLimit:   dailyLimit,
		monthlyLimit: monthlyLimit,
		dailySpent:   decimal.Zero,
		monthlySpent: decimal.Zero,
		version:      1,
		createdAt:    now,
		updatedAt:    now,
	}

	c.domainEvents = append(c.domainEvents, event.NewCardIssued(
		id, tenantID, accountID, cardType.String(), currency, lastFour, now,
	))

	return c, nil
}

// Reconstruct rebuilds a Card aggregate from persisted state.
// No domain events are emitted and no validation is performed beyond construction.
func Reconstruct(
	id, tenantID, accountID uuid.UUID,
	cardType valueobject.CardType,
	status valueobject.CardStatus,
	cardNumber valueobject.CardNumber,
	currency string,
	dailyLimit, monthlyLimit decimal.Decimal,
	dailySpent, monthlySpent decimal.Decimal,
	version int,
	createdAt, updatedAt time.Time,
) Card {
	return Card{
		id:           id,
		tenantID:     tenantID,
		accountID:    accountID,
		cardType:     cardType,
		status:       status,
		cardNumber:   cardNumber,
		currency:     currency,
		dailyLimit:   dailyLimit,
		monthlyLimit: monthlyLimit,
		dailySpent:   dailySpent,
		monthlySpent: monthlySpent,
		version:      version,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// cloneEvents returns a deep copy of the domain events slice so that
// value-receiver methods don't race on the shared backing array.
func (c Card) cloneEvents() []events.DomainEvent {
	if len(c.domainEvents) == 0 {
		return nil
	}
	cloned := make([]events.DomainEvent, len(c.domainEvents))
	copy(cloned, c.domainEvents)
	return cloned
}

// Activate transitions the card from PENDING to ACTIVE.
func (c Card) Activate(now time.Time) (Card, error) {
	if c.status != valueobject.CardStatusPending {
		return c, fmt.Errorf("cannot activate card in %s status, must be PENDING", c.status)
	}

	c.status = valueobject.CardStatusActive
	c.updatedAt = now.UTC()
	c.version++

	c.domainEvents = append(c.cloneEvents(), event.NewCardActivated(
		c.id, c.tenantID, c.accountID, now.UTC(),
	))

	return c, nil
}

// Freeze transitions the card from ACTIVE to FROZEN.
func (c Card) Freeze(now time.Time) (Card, error) {
	if c.status != valueobject.CardStatusActive {
		return c, fmt.Errorf("cannot freeze card in %s status, must be ACTIVE", c.status)
	}

	c.status = valueobject.CardStatusFrozen
	c.updatedAt = now.UTC()
	c.version++

	c.domainEvents = append(c.cloneEvents(), event.NewCardFrozen(
		c.id, c.tenantID, now.UTC(),
	))

	return c, nil
}

// Unfreeze transitions the card from FROZEN back to ACTIVE.
func (c Card) Unfreeze(now time.Time) (Card, error) {
	if c.status != valueobject.CardStatusFrozen {
		return c, fmt.Errorf("cannot unfreeze card in %s status, must be FROZEN", c.status)
	}

	c.status = valueobject.CardStatusActive
	c.updatedAt = now.UTC()
	c.version++

	return c, nil
}

// Cancel transitions the card to CANCELED from any status.
func (c Card) Cancel(now time.Time) (Card, error) {
	if c.status == valueobject.CardStatusCanceled {
		return c, fmt.Errorf("card is already canceled")
	}

	c.status = valueobject.CardStatusCanceled
	c.updatedAt = now.UTC()
	c.version++

	c.domainEvents = append(c.cloneEvents(), event.NewCardCanceled(
		c.id, c.tenantID, now.UTC(),
	))

	return c, nil
}

// AuthorizeTransaction attempts to authorize a transaction against this card.
// It checks status, expiry, and spending limits before approving.
// Returns the updated card, an authorization code, and any error.
func (c Card) AuthorizeTransaction(
	amount decimal.Decimal,
	merchantName, merchantCategory string,
	now time.Time,
) (Card, string, error) {
	if !c.status.IsUsable() {
		c.domainEvents = append(c.cloneEvents(), event.NewTransactionDeclined(
			c.id, c.tenantID, amount, c.currency, merchantName,
			fmt.Sprintf("card is in %s status", c.status), now.UTC(),
		))
		return c, "", fmt.Errorf("card is not usable, current status: %s", c.status)
	}

	if c.cardNumber.IsExpired(now) {
		c.domainEvents = append(c.cloneEvents(), event.NewTransactionDeclined(
			c.id, c.tenantID, amount, c.currency, merchantName,
			"card is expired", now.UTC(),
		))
		return c, "", fmt.Errorf("card is expired")
	}

	if amount.IsNegative() || amount.IsZero() {
		return c, "", fmt.Errorf("transaction amount must be positive")
	}

	newDailySpent := c.dailySpent.Add(amount)
	if newDailySpent.GreaterThan(c.dailyLimit) {
		c.domainEvents = append(c.cloneEvents(), event.NewTransactionDeclined(
			c.id, c.tenantID, amount, c.currency, merchantName,
			"daily spending limit exceeded", now.UTC(),
		))
		return c, "", fmt.Errorf("daily spending limit exceeded: spent %s + %s > limit %s",
			c.dailySpent.String(), amount.String(), c.dailyLimit.String())
	}

	newMonthlySpent := c.monthlySpent.Add(amount)
	if newMonthlySpent.GreaterThan(c.monthlyLimit) {
		c.domainEvents = append(c.cloneEvents(), event.NewTransactionDeclined(
			c.id, c.tenantID, amount, c.currency, merchantName,
			"monthly spending limit exceeded", now.UTC(),
		))
		return c, "", fmt.Errorf("monthly spending limit exceeded: spent %s + %s > limit %s",
			c.monthlySpent.String(), amount.String(), c.monthlyLimit.String())
	}

	c.dailySpent = newDailySpent
	c.monthlySpent = newMonthlySpent
	c.updatedAt = now.UTC()
	c.version++

	authCode := generateAuthCode()

	c.domainEvents = append(c.cloneEvents(), event.NewTransactionAuthorized(
		c.id, c.tenantID, c.accountID, amount, c.currency,
		merchantName, merchantCategory, authCode, now.UTC(),
	))

	return c, authCode, nil
}

// ResetDailySpend resets the daily spending counter.
func (c Card) ResetDailySpend(now time.Time) Card {
	c.dailySpent = decimal.Zero
	c.updatedAt = now.UTC()
	return c
}

// ResetMonthlySpend resets the monthly spending counter.
func (c Card) ResetMonthlySpend(now time.Time) Card {
	c.monthlySpent = decimal.Zero
	c.updatedAt = now.UTC()
	return c
}

// --- Getters ---

func (c Card) ID() uuid.UUID                     { return c.id }
func (c Card) TenantID() uuid.UUID                { return c.tenantID }
func (c Card) AccountID() uuid.UUID               { return c.accountID }
func (c Card) CardType() valueobject.CardType     { return c.cardType }
func (c Card) Status() valueobject.CardStatus     { return c.status }
func (c Card) CardNumber() valueobject.CardNumber { return c.cardNumber }
func (c Card) Currency() string                   { return c.currency }
func (c Card) DailyLimit() decimal.Decimal        { return c.dailyLimit }
func (c Card) MonthlyLimit() decimal.Decimal      { return c.monthlyLimit }
func (c Card) DailySpent() decimal.Decimal        { return c.dailySpent }
func (c Card) MonthlySpent() decimal.Decimal      { return c.monthlySpent }
func (c Card) Version() int                       { return c.version }
func (c Card) CreatedAt() time.Time               { return c.createdAt }
func (c Card) UpdatedAt() time.Time               { return c.updatedAt }

// DomainEvents returns all uncommitted domain events.
func (c Card) DomainEvents() []events.DomainEvent {
	events := make([]events.DomainEvent, len(c.domainEvents))
	copy(events, c.domainEvents)
	return events
}

// ClearEvents returns a new Card with the domain events cleared.
func (c Card) ClearEvents() Card {
	c.domainEvents = nil
	return c
}

// --- Private helpers ---

func generateRandomLastFour() string {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "0000"
	}
	return fmt.Sprintf("%04d", n.Int64())
}

func generateAuthCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			code[i] = '0'
			continue
		}
		code[i] = charset[n.Int64()]
	}
	return string(code)
}
