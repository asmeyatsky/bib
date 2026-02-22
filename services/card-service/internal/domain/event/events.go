package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
)

// DomainEvent is an alias for the shared pkg/events.DomainEvent interface.
type DomainEvent = events.DomainEvent

// CardIssued is emitted when a new card is created.
type CardIssued struct {
	events.BaseEvent
	CardID    uuid.UUID `json:"card_id"`
	AccountID uuid.UUID `json:"account_id"`
	CardType  string    `json:"card_type"`
	Currency  string    `json:"currency"`
	LastFour  string    `json:"last_four"`
	IssuedAt  time.Time `json:"issued_at"`
}

func NewCardIssued(cardID, tenantID, accountID uuid.UUID, cardType, currency, lastFour string, issuedAt time.Time) CardIssued {
	return CardIssued{
		BaseEvent: events.NewBaseEvent("card.issued", cardID.String(), "Card", tenantID.String()),
		CardID:    cardID,
		AccountID: accountID,
		CardType:  cardType,
		Currency:  currency,
		LastFour:  lastFour,
		IssuedAt:  issuedAt,
	}
}

// CardActivated is emitted when a card transitions to ACTIVE status.
type CardActivated struct {
	events.BaseEvent
	CardID      uuid.UUID `json:"card_id"`
	AccountID   uuid.UUID `json:"account_id"`
	ActivatedAt time.Time `json:"activated_at"`
}

func NewCardActivated(cardID, tenantID, accountID uuid.UUID, activatedAt time.Time) CardActivated {
	return CardActivated{
		BaseEvent:   events.NewBaseEvent("card.activated", cardID.String(), "Card", tenantID.String()),
		CardID:      cardID,
		AccountID:   accountID,
		ActivatedAt: activatedAt,
	}
}

// TransactionAuthorized is emitted when a transaction is successfully authorized.
type TransactionAuthorized struct {
	events.BaseEvent
	CardID           uuid.UUID       `json:"card_id"`
	AccountID        uuid.UUID       `json:"account_id"`
	Amount           decimal.Decimal `json:"amount"`
	Currency         string          `json:"currency"`
	MerchantName     string          `json:"merchant_name"`
	MerchantCategory string          `json:"merchant_category"`
	AuthCode         string          `json:"auth_code"`
	AuthorizedAt     time.Time       `json:"authorized_at"`
}

func NewTransactionAuthorized(cardID, tenantID, accountID uuid.UUID, amount decimal.Decimal, currency, merchantName, merchantCategory, authCode string, authorizedAt time.Time) TransactionAuthorized {
	return TransactionAuthorized{
		BaseEvent:        events.NewBaseEvent("card.transaction.authorized", cardID.String(), "Card", tenantID.String()),
		CardID:           cardID,
		AccountID:        accountID,
		Amount:           amount,
		Currency:         currency,
		MerchantName:     merchantName,
		MerchantCategory: merchantCategory,
		AuthCode:         authCode,
		AuthorizedAt:     authorizedAt,
	}
}

// TransactionDeclined is emitted when a transaction is declined.
type TransactionDeclined struct {
	events.BaseEvent
	CardID       uuid.UUID       `json:"card_id"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	MerchantName string          `json:"merchant_name"`
	Reason       string          `json:"reason"`
	DeclinedAt   time.Time       `json:"declined_at"`
}

func NewTransactionDeclined(cardID, tenantID uuid.UUID, amount decimal.Decimal, currency, merchantName, reason string, declinedAt time.Time) TransactionDeclined {
	return TransactionDeclined{
		BaseEvent:    events.NewBaseEvent("card.transaction.declined", cardID.String(), "Card", tenantID.String()),
		CardID:       cardID,
		Amount:       amount,
		Currency:     currency,
		MerchantName: merchantName,
		Reason:       reason,
		DeclinedAt:   declinedAt,
	}
}

// CardFrozen is emitted when a card is frozen.
type CardFrozen struct {
	events.BaseEvent
	CardID   uuid.UUID `json:"card_id"`
	FrozenAt time.Time `json:"frozen_at"`
}

func NewCardFrozen(cardID, tenantID uuid.UUID, frozenAt time.Time) CardFrozen {
	return CardFrozen{
		BaseEvent: events.NewBaseEvent("card.frozen", cardID.String(), "Card", tenantID.String()),
		CardID:    cardID,
		FrozenAt:  frozenAt,
	}
}

// CardCancelled is emitted when a card is cancelled.
type CardCancelled struct {
	events.BaseEvent
	CardID      uuid.UUID `json:"card_id"`
	CancelledAt time.Time `json:"cancelled_at"`
}

func NewCardCancelled(cardID, tenantID uuid.UUID, cancelledAt time.Time) CardCancelled {
	return CardCancelled{
		BaseEvent:   events.NewBaseEvent("card.cancelled", cardID.String(), "Card", tenantID.String()),
		CardID:      cardID,
		CancelledAt: cancelledAt,
	}
}
