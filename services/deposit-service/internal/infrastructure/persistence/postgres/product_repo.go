package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/port"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

// Compile-time interface check.
var _ port.DepositProductRepository = (*ProductRepo)(nil)

// ProductRepo implements DepositProductRepository using PostgreSQL.
type ProductRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{pool: pool}
}

func (r *ProductRepo) Save(ctx context.Context, product model.DepositProduct) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert deposit product
	_, err = tx.Exec(ctx, `
		INSERT INTO deposit_products (id, tenant_id, name, currency, term_days, is_active, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			currency = EXCLUDED.currency,
			term_days = EXCLUDED.term_days,
			is_active = EXCLUDED.is_active,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`, product.ID(), product.TenantID(), product.Name(), product.Currency(),
		product.TermDays(), product.IsActive(), product.Version(),
		product.CreatedAt(), product.UpdatedAt())
	if err != nil {
		return fmt.Errorf("upsert deposit product: %w", err)
	}

	// Delete existing tiers (for upsert scenario)
	_, err = tx.Exec(ctx, `DELETE FROM interest_tiers WHERE product_id = $1`, product.ID())
	if err != nil {
		return fmt.Errorf("delete existing tiers: %w", err)
	}

	// Insert interest tiers
	for i, tier := range product.Tiers() {
		_, err = tx.Exec(ctx, `
			INSERT INTO interest_tiers (product_id, min_balance, max_balance, rate_bps, seq_num)
			VALUES ($1, $2, $3, $4, $5)
		`, product.ID(), tier.MinBalance(), tier.MaxBalance(), tier.RateBps(), i+1)
		if err != nil {
			return fmt.Errorf("insert interest tier %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepo) FindByID(ctx context.Context, id uuid.UUID) (model.DepositProduct, error) {
	var (
		productID uuid.UUID
		tenantID  uuid.UUID
		name      string
		currency  string
		termDays  int
		isActive  bool
		version   int
		createdAt time.Time
		updatedAt time.Time
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, currency, term_days, is_active, version, created_at, updated_at
		FROM deposit_products WHERE id = $1
	`, id).Scan(&productID, &tenantID, &name, &currency, &termDays, &isActive, &version, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.DepositProduct{}, fmt.Errorf("deposit product %s not found", id)
		}
		return model.DepositProduct{}, fmt.Errorf("query deposit product: %w", err)
	}

	// Query interest tiers
	tiers, err := r.findTiersByProductID(ctx, productID)
	if err != nil {
		return model.DepositProduct{}, err
	}

	return model.ReconstructProduct(productID, tenantID, name, currency, tiers, termDays, isActive, version, createdAt, updatedAt), nil
}

func (r *ProductRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.DepositProduct, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM deposit_products WHERE tenant_id = $1 ORDER BY created_at
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query products by tenant: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan product id: %w", err)
		}
		ids = append(ids, id)
	}

	var products []model.DepositProduct
	for _, id := range ids {
		product, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (r *ProductRepo) findTiersByProductID(ctx context.Context, productID uuid.UUID) ([]valueobject.InterestTier, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT min_balance, max_balance, rate_bps
		FROM interest_tiers WHERE product_id = $1 ORDER BY seq_num
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("query interest tiers: %w", err)
	}
	defer rows.Close()

	var tiers []valueobject.InterestTier
	for rows.Next() {
		var (
			minBalance decimal.Decimal
			maxBalance decimal.Decimal
			rateBps    int
		)
		if err := rows.Scan(&minBalance, &maxBalance, &rateBps); err != nil {
			return nil, fmt.Errorf("scan interest tier: %w", err)
		}
		tier, err := valueobject.NewInterestTier(minBalance, maxBalance, rateBps)
		if err != nil {
			return nil, fmt.Errorf("reconstruct interest tier: %w", err)
		}
		tiers = append(tiers, tier)
	}

	return tiers, nil
}
