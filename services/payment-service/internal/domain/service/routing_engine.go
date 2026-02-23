package service

import (
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// RoutingEngine is a domain service that determines the optimal payment rail
// for a given payment based on amount, currency, and destination characteristics.
type RoutingEngine struct{}

// NewRoutingEngine creates a new RoutingEngine instance.
func NewRoutingEngine() *RoutingEngine {
	return &RoutingEngine{}
}

// SelectRail determines the optimal payment rail based on amount, currency,
// whether the transfer is internal, and the destination country.
//
// Routing logic (Phase 1):
//   - Internal transfers -> INTERNAL
//   - USD domestic -> ACH (Phase 1 default for USD)
//   - EUR -> SEPA
//   - International (non-USD, non-EUR, or non-US destination) -> SWIFT
func (e *RoutingEngine) SelectRail(
	_ decimal.Decimal,
	currency string,
	isInternal bool,
	destinationCountry string,
) valueobject.PaymentRail {
	// Internal transfers always use the internal rail.
	if isInternal {
		return valueobject.RailInternal
	}

	// Route based on currency and destination country.
	switch currency {
	case "USD":
		if destinationCountry == "" || destinationCountry == "US" {
			// Phase 1: ACH is the default for domestic USD.
			return valueobject.RailACH
		}
		// USD to non-US destinations go via SWIFT.
		return valueobject.RailSWIFT

	case "EUR":
		if destinationCountry == "" || isEurozone(destinationCountry) {
			return valueobject.RailSEPA
		}
		return valueobject.RailSWIFT

	default:
		// All other currencies route through SWIFT.
		return valueobject.RailSWIFT
	}
}

// isEurozone returns true if the country code belongs to a Eurozone member state.
func isEurozone(country string) bool {
	eurozoneCountries := map[string]bool{
		"AT": true, "BE": true, "CY": true, "DE": true, "EE": true,
		"ES": true, "FI": true, "FR": true, "GR": true, "HR": true,
		"IE": true, "IT": true, "LT": true, "LU": true, "LV": true,
		"MT": true, "NL": true, "PT": true, "SI": true, "SK": true,
	}
	return eurozoneCountries[country]
}
