package adapter

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StubAccountBalanceClient is a stub implementation of the AccountBalanceClient port.
// In production, this would call the ledger-service or account-service via gRPC.
type StubAccountBalanceClient struct {
	logger  *slog.Logger
	balance decimal.Decimal
}

// NewStubAccountBalanceClient creates a new StubAccountBalanceClient
// with a configurable default balance for testing.
func NewStubAccountBalanceClient(logger *slog.Logger, defaultBalance decimal.Decimal) *StubAccountBalanceClient {
	return &StubAccountBalanceClient{
		logger:  logger,
		balance: defaultBalance,
	}
}

// GetAvailableBalance returns the available balance for the given account.
// The stub always returns the configured default balance.
func (c *StubAccountBalanceClient) GetAvailableBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error) {
	c.logger.Info("stub: getting available balance",
		slog.String("account_id", accountID.String()),
		slog.String("balance", c.balance.String()),
	)
	return c.balance, nil
}
