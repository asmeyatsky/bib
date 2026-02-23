package event

import (
	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypeExchangeRate = "ExchangeRate"

// RateUpdated is emitted when an exchange rate is updated.
type RateUpdated struct {
	events.BaseEvent
	Pair           string    `json:"pair"`
	Rate           string    `json:"rate"`
	Provider       string    `json:"provider"`
	ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
}

// NewRateUpdated creates a RateUpdated domain event.
func NewRateUpdated(exchangeRateID, tenantID uuid.UUID, pair, rate, provider string) RateUpdated {
	return RateUpdated{
		BaseEvent:      events.NewBaseEvent("fx.rate.updated", exchangeRateID.String(), AggregateTypeExchangeRate, tenantID.String()),
		ExchangeRateID: exchangeRateID,
		Pair:           pair,
		Rate:           rate,
		Provider:       provider,
	}
}

// RevaluationCompleted is emitted when an FX revaluation run finishes.
type RevaluationCompleted struct {
	events.BaseEvent
	FunctionalCurrency string `json:"functional_currency"`
	TotalGainLoss      string `json:"total_gain_loss"`
	AccountsProcessed  int    `json:"accounts_processed"`
}

// NewRevaluationCompleted creates a RevaluationCompleted domain event.
func NewRevaluationCompleted(tenantID uuid.UUID, functionalCurrency, totalGainLoss string, accountsProcessed int) RevaluationCompleted {
	id := uuid.New()
	return RevaluationCompleted{
		BaseEvent:          events.NewBaseEvent("fx.revaluation.completed", id.String(), "RevaluationRun", tenantID.String()),
		FunctionalCurrency: functionalCurrency,
		TotalGainLoss:      totalGainLoss,
		AccountsProcessed:  accountsProcessed,
	}
}
