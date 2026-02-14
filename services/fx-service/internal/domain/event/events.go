package event

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypeExchangeRate = "ExchangeRate"

// RateUpdated is emitted when an exchange rate is updated.
type RateUpdated struct {
	events.BaseEvent
	ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Pair           string    `json:"pair"`
	Rate           string    `json:"rate"`
	Provider       string    `json:"provider"`
}

// NewRateUpdated creates a RateUpdated domain event.
func NewRateUpdated(exchangeRateID, tenantID uuid.UUID, pair, rate, provider string) RateUpdated {
	payload, _ := json.Marshal(struct {
		ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
		TenantID       uuid.UUID `json:"tenant_id"`
		Pair           string    `json:"pair"`
		Rate           string    `json:"rate"`
		Provider       string    `json:"provider"`
	}{exchangeRateID, tenantID, pair, rate, provider})

	return RateUpdated{
		BaseEvent:      events.NewBaseEvent("fx.rate.updated", exchangeRateID, AggregateTypeExchangeRate, payload),
		ExchangeRateID: exchangeRateID,
		TenantID:       tenantID,
		Pair:           pair,
		Rate:           rate,
		Provider:       provider,
	}
}

// RevaluationCompleted is emitted when an FX revaluation run finishes.
type RevaluationCompleted struct {
	events.BaseEvent
	TenantID           uuid.UUID `json:"tenant_id"`
	FunctionalCurrency string    `json:"functional_currency"`
	TotalGainLoss      string    `json:"total_gain_loss"`
	AccountsProcessed  int       `json:"accounts_processed"`
}

// NewRevaluationCompleted creates a RevaluationCompleted domain event.
func NewRevaluationCompleted(tenantID uuid.UUID, functionalCurrency, totalGainLoss string, accountsProcessed int) RevaluationCompleted {
	id := uuid.New()
	payload, _ := json.Marshal(struct {
		TenantID           uuid.UUID `json:"tenant_id"`
		FunctionalCurrency string    `json:"functional_currency"`
		TotalGainLoss      string    `json:"total_gain_loss"`
		AccountsProcessed  int       `json:"accounts_processed"`
	}{tenantID, functionalCurrency, totalGainLoss, accountsProcessed})

	return RevaluationCompleted{
		BaseEvent:          events.NewBaseEvent("fx.revaluation.completed", id, "RevaluationRun", payload),
		TenantID:           tenantID,
		FunctionalCurrency: functionalCurrency,
		TotalGainLoss:      totalGainLoss,
		AccountsProcessed:  accountsProcessed,
	}
}
