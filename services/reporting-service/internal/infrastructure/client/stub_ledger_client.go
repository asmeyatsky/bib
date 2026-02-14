package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
)

// StubLedgerDataClient is a stub implementation of the LedgerDataClient port.
// In production, this would make gRPC calls to the ledger service.
type StubLedgerDataClient struct{}

// NewStubLedgerDataClient creates a new StubLedgerDataClient.
func NewStubLedgerDataClient() *StubLedgerDataClient {
	return &StubLedgerDataClient{}
}

// GetFinancialData returns sample financial data for development and testing.
func (c *StubLedgerDataClient) GetFinancialData(_ context.Context, tenantID uuid.UUID, period string) (service.ReportData, error) {
	return service.ReportData{
		TenantID:           tenantID,
		Period:             period,
		TotalAssets:        decimal.NewFromInt(1_500_000_000),
		TotalLiabilities:   decimal.NewFromInt(1_350_000_000),
		TotalEquity:        decimal.NewFromInt(150_000_000),
		NetIncome:          decimal.NewFromInt(25_000_000),
		RiskWeightedAssets: decimal.NewFromInt(800_000_000),
		CET1Ratio:          decimal.NewFromFloat(0.1475),
		LCRRatio:           decimal.NewFromFloat(1.2500),
	}, nil
}
