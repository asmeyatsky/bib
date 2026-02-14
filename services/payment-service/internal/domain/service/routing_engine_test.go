package service_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/bibbank/bib/services/payment-service/internal/domain/service"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

func TestRoutingEngine_SelectRail_InternalTransfer(t *testing.T) {
	engine := service.NewRoutingEngine()

	// Internal transfers should always use INTERNAL, regardless of currency or country.
	tests := []struct {
		name     string
		amount   decimal.Decimal
		currency string
		country  string
	}{
		{"USD internal", decimal.NewFromInt(1000), "USD", "US"},
		{"EUR internal", decimal.NewFromInt(500), "EUR", "DE"},
		{"GBP internal", decimal.NewFromInt(250), "GBP", "GB"},
		{"no country internal", decimal.NewFromInt(100), "USD", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rail := engine.SelectRail(tc.amount, tc.currency, true, tc.country)
			assert.Equal(t, valueobject.RailInternal, rail)
		})
	}
}

func TestRoutingEngine_SelectRail_USDDomestic(t *testing.T) {
	engine := service.NewRoutingEngine()

	// USD domestic (US or empty country) should route via ACH.
	rail := engine.SelectRail(decimal.NewFromInt(1000), "USD", false, "US")
	assert.Equal(t, valueobject.RailACH, rail)

	rail = engine.SelectRail(decimal.NewFromInt(5000), "USD", false, "")
	assert.Equal(t, valueobject.RailACH, rail)
}

func TestRoutingEngine_SelectRail_USDInternational(t *testing.T) {
	engine := service.NewRoutingEngine()

	// USD to non-US destinations should route via SWIFT.
	rail := engine.SelectRail(decimal.NewFromInt(1000), "USD", false, "GB")
	assert.Equal(t, valueobject.RailSWIFT, rail)

	rail = engine.SelectRail(decimal.NewFromInt(50000), "USD", false, "JP")
	assert.Equal(t, valueobject.RailSWIFT, rail)
}

func TestRoutingEngine_SelectRail_EUREurozone(t *testing.T) {
	engine := service.NewRoutingEngine()

	// EUR in Eurozone countries should route via SEPA.
	eurozoneCountries := []string{"DE", "FR", "IT", "ES", "NL", "BE", "AT", ""}
	for _, country := range eurozoneCountries {
		t.Run("EUR_"+country, func(t *testing.T) {
			rail := engine.SelectRail(decimal.NewFromInt(500), "EUR", false, country)
			assert.Equal(t, valueobject.RailSEPA, rail)
		})
	}
}

func TestRoutingEngine_SelectRail_EURNonEurozone(t *testing.T) {
	engine := service.NewRoutingEngine()

	// EUR to non-Eurozone countries should route via SWIFT.
	nonEurozoneCountries := []string{"US", "GB", "JP", "CN"}
	for _, country := range nonEurozoneCountries {
		t.Run("EUR_"+country, func(t *testing.T) {
			rail := engine.SelectRail(decimal.NewFromInt(500), "EUR", false, country)
			assert.Equal(t, valueobject.RailSWIFT, rail)
		})
	}
}

func TestRoutingEngine_SelectRail_OtherCurrencies(t *testing.T) {
	engine := service.NewRoutingEngine()

	// Other currencies should always route via SWIFT.
	otherCurrencies := []string{"GBP", "JPY", "CNY", "AUD", "CAD", "CHF"}
	for _, currency := range otherCurrencies {
		t.Run(currency, func(t *testing.T) {
			rail := engine.SelectRail(decimal.NewFromInt(1000), currency, false, "")
			assert.Equal(t, valueobject.RailSWIFT, rail)
		})
	}
}
