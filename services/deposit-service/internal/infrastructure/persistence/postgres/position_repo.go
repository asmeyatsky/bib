package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
)

// Compile-time interface check.
var _ port.DepositPositionRepository = (*PositionRepo)(nil)

// PositionRepo implements DepositPositionRepository using PostgreSQL.
type PositionRepo struct {
	pool *pgxpool.Pool
}

func NewPositionRepo(pool *pgxpool.Pool) *PositionRepo {
	return &PositionRepo{pool: pool}
}

func (r *PositionRepo) Save(ctx context.Context, position model.DepositPosition) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert deposit position
	_, err = tx.Exec(ctx, `
		INSERT INTO deposit_positions (
			id, tenant_id, account_id, product_id, principal, currency,
			accrued_interest, status, opened_at, maturity_date, last_accrual_date,
			version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			accrued_interest = EXCLUDED.accrued_interest,
			status = EXCLUDED.status,
			maturity_date = EXCLUDED.maturity_date,
			last_accrual_date = EXCLUDED.last_accrual_date,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`, position.ID(), position.TenantID(), position.AccountID(), position.ProductID(),
		position.Principal(), position.Currency(), position.AccruedInterest(),
		string(position.Status()), position.OpenedAt(), position.MaturityDate(),
		position.LastAccrualDate(), position.Version(), position.CreatedAt(), position.UpdatedAt())
	if err != nil {
		return fmt.Errorf("upsert deposit position: %w", err)
	}

	// Write domain events to outbox
	for _, evt := range position.DomainEvents() {
		payload, merr := json.Marshal(evt)
		if merr != nil {
			return fmt.Errorf("marshal outbox event: %w", merr)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (id, aggregate_id, aggregate_type, event_type, payload, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, evt.EventID(), evt.AggregateID(), evt.AggregateType(), evt.EventType(), payload, evt.OccurredAt())
		if err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PositionRepo) FindByID(ctx context.Context, id uuid.UUID) (model.DepositPosition, error) {
	return r.scanPosition(ctx, `
		SELECT id, tenant_id, account_id, product_id, principal, currency,
			accrued_interest, status, opened_at, maturity_date, last_accrual_date,
			version, created_at, updated_at
		FROM deposit_positions WHERE id = $1
	`, id)
}

func (r *PositionRepo) FindActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.DepositPosition, error) {
	return r.queryPositions(ctx, `
		SELECT id, tenant_id, account_id, product_id, principal, currency,
			accrued_interest, status, opened_at, maturity_date, last_accrual_date,
			version, created_at, updated_at
		FROM deposit_positions
		WHERE tenant_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at
	`, tenantID)
}

func (r *PositionRepo) FindByAccount(ctx context.Context, accountID uuid.UUID) ([]model.DepositPosition, error) {
	return r.queryPositions(ctx, `
		SELECT id, tenant_id, account_id, product_id, principal, currency,
			accrued_interest, status, opened_at, maturity_date, last_accrual_date,
			version, created_at, updated_at
		FROM deposit_positions
		WHERE account_id = $1
		ORDER BY created_at
	`, accountID)
}

func (r *PositionRepo) scanPosition(ctx context.Context, query string, args ...interface{}) (model.DepositPosition, error) {
	var (
		id              uuid.UUID
		tenantID        uuid.UUID
		accountID       uuid.UUID
		productID       uuid.UUID
		principal       decimal.Decimal
		currency        string
		accruedInterest decimal.Decimal
		status          string
		openedAt        time.Time
		maturityDate    *time.Time
		lastAccrualDate time.Time
		version         int
		createdAt       time.Time
		updatedAt       time.Time
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id, &tenantID, &accountID, &productID, &principal, &currency,
		&accruedInterest, &status, &openedAt, &maturityDate, &lastAccrualDate,
		&version, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.DepositPosition{}, fmt.Errorf("deposit position not found")
		}
		return model.DepositPosition{}, fmt.Errorf("query deposit position: %w", err)
	}

	return model.ReconstructPosition(
		id, tenantID, accountID, productID, principal, currency,
		accruedInterest, model.PositionStatus(status), openedAt, maturityDate,
		lastAccrualDate, version, createdAt, updatedAt,
	), nil
}

func (r *PositionRepo) queryPositions(ctx context.Context, query string, args ...interface{}) ([]model.DepositPosition, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query positions: %w", err)
	}
	defer rows.Close()

	var positions []model.DepositPosition
	for rows.Next() {
		var (
			id              uuid.UUID
			tenantID        uuid.UUID
			accountID       uuid.UUID
			productID       uuid.UUID
			principal       decimal.Decimal
			currency        string
			accruedInterest decimal.Decimal
			status          string
			openedAt        time.Time
			maturityDate    *time.Time
			lastAccrualDate time.Time
			version         int
			createdAt       time.Time
			updatedAt       time.Time
		)

		if err := rows.Scan(
			&id, &tenantID, &accountID, &productID, &principal, &currency,
			&accruedInterest, &status, &openedAt, &maturityDate, &lastAccrualDate,
			&version, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deposit position: %w", err)
		}

		positions = append(positions, model.ReconstructPosition(
			id, tenantID, accountID, productID, principal, currency,
			accruedInterest, model.PositionStatus(status), openedAt, maturityDate,
			lastAccrualDate, version, createdAt, updatedAt,
		))
	}

	return positions, nil
}
