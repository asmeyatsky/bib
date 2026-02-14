package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiRailRouter_USDDomestic_ReturnsACHAndFedNow(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(5000)

	options, err := router.FindOptimalRoute(amount, "USD", "US", false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(options), 2)

	railNames := make(map[string]bool)
	for _, opt := range options {
		railNames[opt.Rail] = true
	}
	assert.True(t, railNames["ACH"], "expected ACH to be available for USD domestic")
	assert.True(t, railNames["FEDNOW"], "expected FEDNOW to be available for USD domestic")
}

func TestMultiRailRouter_EURToGermany_ReturnsSEPA(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(10000)

	options, err := router.FindOptimalRoute(amount, "EUR", "DE", false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(options), 1)

	// SEPA should be the top-ranked option for EUR to Germany
	assert.Equal(t, "SEPA", options[0].Rail)
}

func TestMultiRailRouter_USDInternational_ReturnsSWIFT(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(100000)

	options, err := router.FindOptimalRoute(amount, "USD", "GB", false)
	require.NoError(t, err)
	require.Len(t, options, 1)
	assert.Equal(t, "SWIFT", options[0].Rail)
}

func TestMultiRailRouter_PreferSpeed_RanksFedNowHigher(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(5000)

	options, err := router.FindOptimalRoute(amount, "USD", "US", true)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(options), 2)

	// With preferSpeed=true, FedNow (10s) should rank higher than ACH (86400s)
	fedNowIdx := -1
	achIdx := -1
	for i, opt := range options {
		if opt.Rail == "FEDNOW" {
			fedNowIdx = i
		}
		if opt.Rail == "ACH" {
			achIdx = i
		}
	}
	require.NotEqual(t, -1, fedNowIdx, "FEDNOW should be in options")
	require.NotEqual(t, -1, achIdx, "ACH should be in options")
	assert.Less(t, fedNowIdx, achIdx, "FEDNOW should be ranked higher (lower index) than ACH when preferring speed")
}

func TestMultiRailRouter_AmountExceedsMax_ExcludesRail(t *testing.T) {
	router := NewMultiRailRouter()
	// FedNow max is 500,000 - use an amount that exceeds it but is under ACH max (1,000,000)
	amount := decimal.NewFromInt(750000)

	options, err := router.FindOptimalRoute(amount, "USD", "US", false)
	require.NoError(t, err)

	for _, opt := range options {
		assert.NotEqual(t, "FEDNOW", opt.Rail, "FEDNOW should be excluded when amount exceeds its max")
	}
}

func TestMultiRailRouter_NoAvailableRail_ReturnsError(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(1000)

	// INR to India - no rail supports this combination
	_, err := router.FindOptimalRoute(amount, "INR", "IN", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available payment rail")
}

func TestMultiRailRouter_USDDomestic_IncludesCHIPS(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(5000)

	options, err := router.FindOptimalRoute(amount, "USD", "US", false)
	require.NoError(t, err)

	railNames := make(map[string]bool)
	for _, opt := range options {
		railNames[opt.Rail] = true
	}
	assert.True(t, railNames["CHIPS"], "expected CHIPS to be available for USD domestic")
}

func TestMultiRailRouter_ScoreCalculation_CostVsSpeed(t *testing.T) {
	router := NewMultiRailRouter()
	amount := decimal.NewFromInt(5000)

	// Without speed preference, cost-weighted scoring
	optsCost, err := router.FindOptimalRoute(amount, "USD", "US", false)
	require.NoError(t, err)

	// With speed preference
	optsSpeed, err := router.FindOptimalRoute(amount, "USD", "US", true)
	require.NoError(t, err)

	// The order should differ between cost-optimized and speed-optimized
	// ACH has low cost (5 bps) but slow speed (86400s)
	// FedNow has higher cost (10 bps) but fast speed (10s)
	// When preferring speed, FedNow should be ranked first
	assert.Equal(t, "FEDNOW", optsSpeed[0].Rail, "FedNow should be first when preferring speed")

	// Verify scores are properly ordered
	for i := 1; i < len(optsCost); i++ {
		assert.LessOrEqual(t, optsCost[i-1].Score, optsCost[i].Score, "options should be sorted by score ascending")
	}
	for i := 1; i < len(optsSpeed); i++ {
		assert.LessOrEqual(t, optsSpeed[i-1].Score, optsSpeed[i].Score, "options should be sorted by score ascending")
	}
}
