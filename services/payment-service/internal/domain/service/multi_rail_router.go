package service

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// MultiRailRouter selects the optimal payment rail based on cost, speed, and availability.
type MultiRailRouter struct {
	railConfigs map[string]RailConfig
}

type RailConfig struct {
	Rail         valueobject.PaymentRail
	MaxAmount    decimal.Decimal
	Currencies   []string
	CostBps      int // cost in basis points
	SpeedSeconds int // typical settlement time
	Available    bool
	Countries    []string // supported destination countries
}

func NewMultiRailRouter() *MultiRailRouter {
	configs := map[string]RailConfig{
		"ACH": {
			Rail:         valueobject.RailACH,
			MaxAmount:    decimal.NewFromInt(1000000),
			Currencies:   []string{"USD"},
			CostBps:      5,
			SpeedSeconds: 86400, // 1 day
			Available:    true,
			Countries:    []string{"US"},
		},
		"FEDNOW": {
			Rail:         valueobject.RailFedNow,
			MaxAmount:    decimal.NewFromInt(500000),
			Currencies:   []string{"USD"},
			CostBps:      10,
			SpeedSeconds: 10, // near instant
			Available:    true,
			Countries:    []string{"US"},
		},
		"SWIFT": {
			Rail:         valueobject.RailSWIFT,
			MaxAmount:    decimal.NewFromInt(50000000),
			Currencies:   []string{"USD", "EUR", "GBP", "JPY", "CHF"},
			CostBps:      25,
			SpeedSeconds: 172800, // 2 days
			Available:    true,
			Countries:    []string{}, // international - all countries
		},
		"SEPA": {
			Rail:         valueobject.RailSEPA,
			MaxAmount:    decimal.NewFromInt(999999999),
			Currencies:   []string{"EUR"},
			CostBps:      3,
			SpeedSeconds: 3600, // 1 hour for SEPA Instant
			Available:    true,
			Countries:    []string{"DE", "FR", "IT", "ES", "NL", "BE", "AT", "PT", "IE", "FI", "GR", "LU"},
		},
		"CHIPS": {
			Rail:         valueobject.RailCHIPS,
			MaxAmount:    decimal.NewFromInt(999999999),
			Currencies:   []string{"USD"},
			CostBps:      15,
			SpeedSeconds: 7200, // same day
			Available:    true,
			Countries:    []string{"US"},
		},
	}
	return &MultiRailRouter{railConfigs: configs}
}

// RouteOption represents a possible route for a payment.
type RouteOption struct {
	Rail         string
	CostBps      int
	SpeedSeconds int
	Score        float64 // lower is better
}

// FindOptimalRoute returns ranked route options for a payment.
func (r *MultiRailRouter) FindOptimalRoute(amount decimal.Decimal, currency string, destCountry string, preferSpeed bool) ([]RouteOption, error) {
	var options []RouteOption
	for name, cfg := range r.railConfigs {
		if !cfg.Available {
			continue
		}
		if amount.GreaterThan(cfg.MaxAmount) {
			continue
		}
		currencyMatch := false
		for _, c := range cfg.Currencies {
			if c == currency {
				currencyMatch = true
				break
			}
		}
		if !currencyMatch {
			continue
		}
		// Country check (empty = international)
		if len(cfg.Countries) > 0 {
			countryMatch := false
			for _, c := range cfg.Countries {
				if c == destCountry {
					countryMatch = true
					break
				}
			}
			if !countryMatch {
				continue
			}
		}

		// Calculate score (weighted combination of cost and speed)
		costWeight := 0.6
		speedWeight := 0.4
		if preferSpeed {
			costWeight = 0.3
			speedWeight = 0.7
		}
		score := costWeight*float64(cfg.CostBps) + speedWeight*float64(cfg.SpeedSeconds)/3600.0

		options = append(options, RouteOption{
			Rail:         name,
			CostBps:      cfg.CostBps,
			SpeedSeconds: cfg.SpeedSeconds,
			Score:        score,
		})
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("no available payment rail for %s %s to %s", amount, currency, destCountry)
	}
	// Sort by score (lowest first)
	sort.Slice(options, func(i, j int) bool { return options[i].Score < options[j].Score })
	return options, nil
}
