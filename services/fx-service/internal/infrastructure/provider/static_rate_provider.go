package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// staticRates maps "BASE/QUOTE" to a rate decimal.
var staticRates = map[string]decimal.Decimal{
	"EUR/USD": decimal.RequireFromString("1.0850"),
	"GBP/USD": decimal.RequireFromString("1.2650"),
	"USD/JPY": decimal.RequireFromString("149.50"),
	"USD/CHF": decimal.RequireFromString("0.8820"),
	"AUD/USD": decimal.RequireFromString("0.6520"),
	"USD/CAD": decimal.RequireFromString("1.3580"),
	"NZD/USD": decimal.RequireFromString("0.6080"),
	"EUR/GBP": decimal.RequireFromString("0.8580"),
	"EUR/JPY": decimal.RequireFromString("162.20"),
	"GBP/JPY": decimal.RequireFromString("189.10"),
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

	if d, ok := staticRates[key]; ok {
		return valueobject.NewSpotRate(d)
	}

	// Try the inverse pair.
	inverseKey := strings.ToUpper(quote) + "/" + strings.ToUpper(base)
	if d, ok := staticRates[inverseKey]; ok {
		rate, err := valueobject.NewSpotRate(d)
		if err != nil {
			return valueobject.SpotRate{}, err
		}
		return rate.Inverse(), nil
	}

	return valueobject.SpotRate{}, fmt.Errorf("no static rate available for %s", key)
}
