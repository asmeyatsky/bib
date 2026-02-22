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

	"github.com/bibbank/bib/services/fx-service/internal/domain/model"
	"github.com/bibbank/bib/services/fx-service/internal/domain/port"
	"github.com/bibbank/bib/services/fx-service/internal/domain/valueobject"
)

// Compile-time interface check.
var _ port.ExchangeRateRepository = (*ExchangeRateRepo)(nil)

// ExchangeRateRepo implements ExchangeRateRepository using PostgreSQL.
type ExchangeRateRepo struct {
	pool *pgxpool.Pool
}

// NewExchangeRateRepo creates a new ExchangeRateRepo.
func NewExchangeRateRepo(pool *pgxpool.Pool) *ExchangeRateRepo {
	return &ExchangeRateRepo{pool: pool}
}

// Save persists an exchange rate, writing domain events to the outbox in the same transaction.
func (r *ExchangeRateRepo) Save(ctx context.Context, rate model.ExchangeRate) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO exchange_rates (id, tenant_id, base_currency, quote_currency, rate, inverse_rate, provider, effective_at, expires_at, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			rate = EXCLUDED.rate,
			inverse_rate = EXCLUDED.inverse_rate,
			provider = EXCLUDED.provider,
			effective_at = EXCLUDED.effective_at,
			expires_at = EXCLUDED.expires_at,
			version = EXCLUDED.version
	`, rate.ID(), rate.TenantID(), rate.Pair().Base(), rate.Pair().Quote(),
		rate.Rate().Rate(), rate.InverseRate().Rate(), rate.Provider(),
		rate.EffectiveAt(), rate.ExpiresAt(), rate.Version(), rate.CreatedAt())
	if err != nil {
		return fmt.Errorf("upsert exchange rate: %w", err)
	}

	// Write domain events to outbox.
	for _, evt := range rate.DomainEvents() {
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

// FindByPair retrieves the latest exchange rate for a currency pair within a tenant.
func (r *ExchangeRateRepo) FindByPair(ctx context.Context, tenantID uuid.UUID, pair valueobject.CurrencyPair) (model.ExchangeRate, error) {
	return r.scanOne(ctx, `
		SELECT id, tenant_id, base_currency, quote_currency, rate, inverse_rate, provider, effective_at, expires_at, version, created_at
		FROM exchange_rates
		WHERE tenant_id = $1 AND base_currency = $2 AND quote_currency = $3
		ORDER BY effective_at DESC
		LIMIT 1
	`, tenantID, pair.Base(), pair.Quote())
}

// FindLatest retrieves the most recent exchange rate for a pair across all tenants.
func (r *ExchangeRateRepo) FindLatest(ctx context.Context, pair valueobject.CurrencyPair) (model.ExchangeRate, error) {
	return r.scanOne(ctx, `
		SELECT id, tenant_id, base_currency, quote_currency, rate, inverse_rate, provider, effective_at, expires_at, version, created_at
		FROM exchange_rates
		WHERE base_currency = $1 AND quote_currency = $2
		ORDER BY effective_at DESC
		LIMIT 1
	`, pair.Base(), pair.Quote())
}

// ListByBase returns all exchange rates with the given base currency for a tenant.
func (r *ExchangeRateRepo) ListByBase(ctx context.Context, tenantID uuid.UUID, baseCurrency string, asOf time.Time) ([]model.ExchangeRate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (quote_currency) id, tenant_id, base_currency, quote_currency, rate, inverse_rate, provider, effective_at, expires_at, version, created_at
		FROM exchange_rates
		WHERE tenant_id = $1 AND base_currency = $2 AND effective_at <= $3
		ORDER BY quote_currency, effective_at DESC
	`, tenantID, baseCurrency, asOf)
	if err != nil {
		return nil, fmt.Errorf("query exchange rates by base: %w", err)
	}
	defer rows.Close()

	var rates []model.ExchangeRate
	for rows.Next() {
		rate, err := scanExchangeRate(rows)
		if err != nil {
			return nil, err
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

// scanOne executes a query that is expected to return at most one row.
func (r *ExchangeRateRepo) scanOne(ctx context.Context, query string, args ...interface{}) (model.ExchangeRate, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("query exchange rate: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return model.ExchangeRate{}, fmt.Errorf("exchange rate not found")
	}

	return scanExchangeRate(rows)
}

// scanExchangeRate reads one ExchangeRate from a pgx.Rows cursor.
func scanExchangeRate(rows pgx.Rows) (model.ExchangeRate, error) {
	var (
		id            uuid.UUID
		tenantID      uuid.UUID
		baseCurrency  string
		quoteCurrency string
		rate          decimal.Decimal
		inverseRate   decimal.Decimal
		provider      string
		effectiveAt   time.Time
		expiresAt     time.Time
		version       int
		createdAt     time.Time
	)

	err := rows.Scan(&id, &tenantID, &baseCurrency, &quoteCurrency, &rate, &inverseRate,
		&provider, &effectiveAt, &expiresAt, &version, &createdAt)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("scan exchange rate: %w", err)
	}

	pair, err := valueobject.NewCurrencyPair(baseCurrency, quoteCurrency)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("reconstruct currency pair: %w", err)
	}

	spotRate, err := valueobject.NewSpotRate(rate)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("reconstruct spot rate: %w", err)
	}

	invRate, err := valueobject.NewSpotRate(inverseRate)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("reconstruct inverse rate: %w", err)
	}

	return model.Reconstruct(id, tenantID, pair, spotRate, invRate, provider, effectiveAt, expiresAt, version, createdAt), nil
}
