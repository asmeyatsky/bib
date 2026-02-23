package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

var _ port.BalanceRepository = (*BalanceRepo)(nil)

// BalanceRepo implements BalanceRepository using PostgreSQL.
type BalanceRepo struct {
	pool *pgxpool.Pool
}

func NewBalanceRepo(pool *pgxpool.Pool) *BalanceRepo {
	return &BalanceRepo{pool: pool}
}

func (r *BalanceRepo) GetBalance(ctx context.Context, accountCode valueobject.AccountCode, currency string, _ time.Time) (decimal.Decimal, error) {
	var balance decimal.Decimal
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(balance, 0) FROM account_balances
		WHERE account_code = $1 AND currency = $2
	`, accountCode.Code(), currency).Scan(&balance)
	if err != nil {
		// If no row, balance is zero
		return decimal.Zero, nil
	}
	return balance, nil
}

func (r *BalanceRepo) UpdateBalance(ctx context.Context, accountCode valueobject.AccountCode, currency string, delta decimal.Decimal) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO account_balances (account_code, currency, balance, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_code, currency) DO UPDATE SET
			balance = account_balances.balance + EXCLUDED.balance,
			updated_at = EXCLUDED.updated_at
	`, accountCode.Code(), currency, delta, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	return nil
}
