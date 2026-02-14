package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// IssueCardRequest is the input DTO for issuing a new card.
type IssueCardRequest struct {
	TenantID     uuid.UUID       `json:"tenant_id"`
	AccountID    uuid.UUID       `json:"account_id"`
	CardType     string          `json:"card_type"`
	Currency     string          `json:"currency"`
	DailyLimit   decimal.Decimal `json:"daily_limit"`
	MonthlyLimit decimal.Decimal `json:"monthly_limit"`
}

// IssueCardResponse is the output DTO after issuing a card.
type IssueCardResponse struct {
	CardID      uuid.UUID `json:"card_id"`
	LastFour    string    `json:"last_four"`
	ExpiryMonth string    `json:"expiry_month"`
	ExpiryYear  string    `json:"expiry_year"`
	Status      string    `json:"status"`
	CardType    string    `json:"card_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuthorizeTransactionRequest is the input DTO for authorizing a card transaction.
type AuthorizeTransactionRequest struct {
	CardID           uuid.UUID       `json:"card_id"`
	Amount           decimal.Decimal `json:"amount"`
	Currency         string          `json:"currency"`
	MerchantName     string          `json:"merchant_name"`
	MerchantCategory string          `json:"merchant_category"`
}

// AuthorizeTransactionResponse is the output DTO after transaction authorization.
type AuthorizeTransactionResponse struct {
	Approved bool   `json:"approved"`
	AuthCode string `json:"auth_code,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// GetCardRequest is the input DTO for retrieving a card.
type GetCardRequest struct {
	CardID uuid.UUID `json:"card_id"`
}

// CardResponse is the general output DTO for card details.
type CardResponse struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	AccountID    uuid.UUID       `json:"account_id"`
	CardType     string          `json:"card_type"`
	Status       string          `json:"status"`
	LastFour     string          `json:"last_four"`
	ExpiryMonth  string          `json:"expiry_month"`
	ExpiryYear   string          `json:"expiry_year"`
	Currency     string          `json:"currency"`
	DailyLimit   decimal.Decimal `json:"daily_limit"`
	MonthlyLimit decimal.Decimal `json:"monthly_limit"`
	DailySpent   decimal.Decimal `json:"daily_spent"`
	MonthlySpent decimal.Decimal `json:"monthly_spent"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// FreezeCardRequest is the input DTO for freezing a card.
type FreezeCardRequest struct {
	CardID uuid.UUID `json:"card_id"`
}

// FreezeCardResponse is the output DTO after freezing a card.
type FreezeCardResponse struct {
	CardID uuid.UUID `json:"card_id"`
	Status string    `json:"status"`
}
