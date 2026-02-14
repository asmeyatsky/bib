package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

var _ port.FiscalPeriodRepository = (*FiscalPeriodRepo)(nil)

type FiscalPeriodRepo struct {
	pool *pgxpool.Pool
}

func NewFiscalPeriodRepo(pool *pgxpool.Pool) *FiscalPeriodRepo {
	return &FiscalPeriodRepo{pool: pool}
}

func (r *FiscalPeriodRepo) GetPeriodStatus(ctx context.Context, tenantID uuid.UUID, period valueobject.FiscalPeriod) (valueobject.PeriodStatus, error) {
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT status FROM fiscal_periods
		WHERE tenant_id = $1 AND year = $2 AND month = $3
	`, tenantID, period.Year(), int(period.Month())).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return valueobject.PeriodStatusOpen, nil
		}
		return "", fmt.Errorf("get period status: %w", err)
	}
	return valueobject.PeriodStatus(status), nil
}

func (r *FiscalPeriodRepo) ClosePeriod(ctx context.Context, tenantID uuid.UUID, period valueobject.FiscalPeriod) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fiscal_periods (tenant_id, year, month, status, closed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, year, month) DO UPDATE SET
			status = EXCLUDED.status,
			closed_at = EXCLUDED.closed_at
	`, tenantID, period.Year(), int(period.Month()), string(valueobject.PeriodStatusClosed), now)
	if err != nil {
		return fmt.Errorf("close period: %w", err)
	}
	return nil
}
