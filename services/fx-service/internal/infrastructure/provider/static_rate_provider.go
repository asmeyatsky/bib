package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// staticRates maps "BASE/QUOTE" to a rate decimal string.
var staticRates = map[string]string{
	"EUR/USD": "1.0850",
	"GBP/USD": "1.2650",
	"USD/JPY": "149.50",
	"USD/CHF": "0.8820",
	"AUD/USD": "0.6520",
	"USD/CAD": "1.3580",
	"NZD/USD": "0.6080",
	"EUR/GBP": "0.8580",
	"EUR/JPY": "162.20",
	"GBP/JPY": "189.10",
}

// StaticRateProvider returns hardcoded exchange rates for common currency pairs.
// It is intended for development, testing, and CI environments.
type StaticRateProvider struct{}

// NewStaticRateProvider creates a new StaticRateProvider.
func NewStaticRateProvider() *StaticRateProvider {
	return &StaticRateProvider{}
}

// FetchRate returns a static rate for the given currency pair.
func (p *StaticRateProvider) FetchRate(_ context.Context, base, quote string) (valueobject.SpotRate, error) {
	key := strings.ToUpper(base) + "/" + strings.ToUpper(quote)

	if rateStr, ok := staticRates[key]; ok {
		d, _ := decimal.NewFromString(rateStr)
		return valueobject.NewSpotRate(d)
	}

	// Try the inverse pair.
	inverseKey := strings.ToUpper(quote) + "/" + strings.ToUpper(base)
	if rateStr, ok := staticRates[inverseKey]; ok {
		d, _ := decimal.NewFromString(rateStr)
		rate, err := valueobject.NewSpotRate(d)
		if err != nil {
			return valueobject.SpotRate{}, err
		}
		return rate.Inverse(), nil
	}

	return valueobject.SpotRate{}, fmt.Errorf("no static rate available for %s", key)
}
