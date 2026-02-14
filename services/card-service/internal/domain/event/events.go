package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DomainEvent is the marker interface for all domain events.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// CardIssued is emitted when a new card is created.
type CardIssued struct {
	CardID    uuid.UUID `json:"card_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	AccountID uuid.UUID `json:"account_id"`
	CardType  string    `json:"card_type"`
	Currency  string    `json:"currency"`
	LastFour  string    `json:"last_four"`
	IssuedAt  time.Time `json:"issued_at"`
}

func (e CardIssued) EventType() string    { return "card.issued" }
func (e CardIssued) OccurredAt() time.Time { return e.IssuedAt }

// CardActivated is emitted when a card transitions to ACTIVE status.
type CardActivated struct {
	CardID      uuid.UUID `json:"card_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	AccountID   uuid.UUID `json:"account_id"`
	ActivatedAt time.Time `json:"activated_at"`
}

func (e CardActivated) EventType() string    { return "card.activated" }
func (e CardActivated) OccurredAt() time.Time { return e.ActivatedAt }

// TransactionAuthorized is emitted when a transaction is successfully authorized.
type TransactionAuthorized struct {
	CardID           uuid.UUID       `json:"card_id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	AccountID        uuid.UUID       `json:"account_id"`
	Amount           decimal.Decimal `json:"amount"`
	Currency         string          `json:"currency"`
	MerchantName     string          `json:"merchant_name"`
	MerchantCategory string          `json:"merchant_category"`
	AuthCode         string          `json:"auth_code"`
	AuthorizedAt     time.Time       `json:"authorized_at"`
}

func (e TransactionAuthorized) EventType() string    { return "card.transaction.authorized" }
func (e TransactionAuthorized) OccurredAt() time.Time { return e.AuthorizedAt }

// TransactionDeclined is emitted when a transaction is declined.
type TransactionDeclined struct {
	CardID       uuid.UUID       `json:"card_id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	MerchantName string          `json:"merchant_name"`
	Reason       string          `json:"reason"`
	DeclinedAt   time.Time       `json:"declined_at"`
}

func (e TransactionDeclined) EventType() string    { return "card.transaction.declined" }
func (e TransactionDeclined) OccurredAt() time.Time { return e.DeclinedAt }

// CardFrozen is emitted when a card is frozen.
type CardFrozen struct {
	CardID   uuid.UUID `json:"card_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	FrozenAt time.Time `json:"frozen_at"`
}

func (e CardFrozen) EventType() string    { return "card.frozen" }
func (e CardFrozen) OccurredAt() time.Time { return e.FrozenAt }

// CardCancelled is emitted when a card is cancelled.
type CardCancelled struct {
	CardID      uuid.UUID `json:"card_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	CancelledAt time.Time `json:"cancelled_at"`
}

func (e CardCancelled) EventType() string    { return "card.cancelled" }
func (e CardCancelled) OccurredAt() time.Time { return e.CancelledAt }
