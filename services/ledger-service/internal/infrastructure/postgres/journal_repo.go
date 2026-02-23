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

	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// Compile-time interface check
var _ port.JournalRepository = (*JournalRepo)(nil)

// JournalRepo implements JournalRepository using PostgreSQL.
type JournalRepo struct {
	pool *pgxpool.Pool
}

func NewJournalRepo(pool *pgxpool.Pool) *JournalRepo {
	return &JournalRepo{pool: pool}
}

func (r *JournalRepo) Save(ctx context.Context, entry model.JournalEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	//nolint:errcheck
	defer tx.Rollback(ctx)

	// Upsert journal entry
	_, err = tx.Exec(ctx, `
		INSERT INTO journal_entries (id, tenant_id, effective_date, status, description, reference, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			effective_date = EXCLUDED.effective_date,
			status = EXCLUDED.status,
			description = EXCLUDED.description,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`, entry.ID(), entry.TenantID(), entry.EffectiveDate(), string(entry.Status()),
		entry.Description(), entry.Reference(), entry.Version(), entry.CreatedAt(), entry.UpdatedAt())
	if err != nil {
		return fmt.Errorf("upsert journal entry: %w", err)
	}

	// Delete existing postings (for upsert scenario)
	_, err = tx.Exec(ctx, `DELETE FROM posting_pairs WHERE entry_id = $1`, entry.ID())
	if err != nil {
		return fmt.Errorf("delete existing postings: %w", err)
	}

	// Insert posting pairs
	for i, p := range entry.Postings() {
		_, err = tx.Exec(ctx, `
			INSERT INTO posting_pairs (entry_id, debit_account, credit_account, amount, currency, description, seq_num)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, entry.ID(), p.DebitAccount().Code(), p.CreditAccount().Code(),
			p.Amount(), p.Currency(), p.Description(), i+1)
		if err != nil {
			return fmt.Errorf("insert posting pair %d: %w", i, err)
		}
	}

	// Write domain events to outbox
	for _, evt := range entry.DomainEvents() {
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

func (r *JournalRepo) FindByID(ctx context.Context, id uuid.UUID) (model.JournalEntry, error) {
	// Query journal entry
	var (
		entryID       uuid.UUID
		tenantID      uuid.UUID
		effectiveDate time.Time
		status        string
		description   string
		reference     string
		version       int
		createdAt     time.Time
		updatedAt     time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, effective_date, status, description, reference, version, created_at, updated_at
		FROM journal_entries WHERE id = $1
	`, id).Scan(&entryID, &tenantID, &effectiveDate, &status, &description, &reference, &version, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.JournalEntry{}, fmt.Errorf("journal entry %s not found", id)
		}
		return model.JournalEntry{}, fmt.Errorf("query journal entry: %w", err)
	}

	// Query posting pairs
	rows, err := r.pool.Query(ctx, `
		SELECT debit_account, credit_account, amount, currency, description
		FROM posting_pairs WHERE entry_id = $1 ORDER BY seq_num
	`, id)
	if err != nil {
		return model.JournalEntry{}, fmt.Errorf("query posting pairs: %w", err)
	}
	defer rows.Close()

	var postings []valueobject.PostingPair
	for rows.Next() {
		var (
			debitStr, creditStr, currency, desc string
			amount                              decimal.Decimal
		)
		if err := rows.Scan(&debitStr, &creditStr, &amount, &currency, &desc); err != nil {
			return model.JournalEntry{}, fmt.Errorf("scan posting pair: %w", err)
		}
		debit, debitErr := valueobject.NewAccountCode(debitStr)
		if debitErr != nil {
			return model.JournalEntry{}, fmt.Errorf("invalid debit account code %q: %w", debitStr, debitErr)
		}
		credit, creditErr := valueobject.NewAccountCode(creditStr)
		if creditErr != nil {
			return model.JournalEntry{}, fmt.Errorf("invalid credit account code %q: %w", creditStr, creditErr)
		}
		pair, pairErr := valueobject.NewPostingPair(debit, credit, amount, currency, desc)
		if pairErr != nil {
			return model.JournalEntry{}, fmt.Errorf("invalid posting pair: %w", pairErr)
		}
		postings = append(postings, pair)
	}

	return model.Reconstruct(entryID, tenantID, effectiveDate, postings, model.EntryStatus(status), description, reference, version, createdAt, updatedAt), nil
}

func (r *JournalRepo) ListByAccount(ctx context.Context, tenantID uuid.UUID, accountCode valueobject.AccountCode, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
	// Count total
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT je.id) FROM journal_entries je
		JOIN posting_pairs pp ON pp.entry_id = je.id
		WHERE je.tenant_id = $1
		AND (pp.debit_account = $2 OR pp.credit_account = $2)
		AND je.effective_date >= $3 AND je.effective_date <= $4
	`, tenantID, accountCode.Code(), from, to).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count entries: %w", err)
	}

	// Query entries
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT je.id FROM journal_entries je
		JOIN posting_pairs pp ON pp.entry_id = je.id
		WHERE je.tenant_id = $1
		AND (pp.debit_account = $2 OR pp.credit_account = $2)
		AND je.effective_date >= $3 AND je.effective_date <= $4
		ORDER BY je.id
		LIMIT $5 OFFSET $6
	`, tenantID, accountCode.Code(), from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan entry id: %w", err)
		}
		ids = append(ids, id)
	}

	var entries []model.JournalEntry
	for _, id := range ids {
		entry, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

func (r *JournalRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM journal_entries
		WHERE tenant_id = $1 AND effective_date >= $2 AND effective_date <= $3
	`, tenantID, from, to).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count entries: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id FROM journal_entries
		WHERE tenant_id = $1 AND effective_date >= $2 AND effective_date <= $3
		ORDER BY effective_date DESC, id
		LIMIT $4 OFFSET $5
	`, tenantID, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan entry id: %w", err)
		}
		ids = append(ids, id)
	}

	var entries []model.JournalEntry
	for _, id := range ids {
		entry, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}
