//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/testutil"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
	"github.com/bibbank/bib/services/ledger-service/internal/infrastructure/postgres"
)

func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "internal", "infrastructure", "postgres", "migrations")
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pg := testutil.NewPostgresContainer(ctx, t)
	t.Cleanup(func() { pg.Cleanup(t) })

	pg.RunMigrations(t, migrationsDir())

	// Disable RLS for test queries by making the connection a superuser.
	// The testuser is already the owner, so RLS policies don't apply to table owners
	// unless FORCE ROW LEVEL SECURITY is set. We bypass by setting the tenant explicitly.
	return pg.Pool
}

func setTenant(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "SET app.tenant_id = '"+tenantID.String()+"'")
	require.NoError(t, err)
}

func newTestEntry(t *testing.T, tenantID uuid.UUID) model.JournalEntry {
	t.Helper()
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(1000)

	posting, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "Test posting")
	require.NoError(t, err)

	entry, err := model.NewJournalEntry(
		tenantID,
		time.Now().UTC().Truncate(time.Microsecond),
		[]valueobject.PostingPair{posting},
		"Test journal entry",
		"REF-001",
	)
	require.NoError(t, err)
	return entry
}

func TestJournalRepository_SaveAndGet(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewJournalRepo(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	entry := newTestEntry(t, tenantID)

	// Save the journal entry.
	err := repo.Save(ctx, entry)
	require.NoError(t, err)

	// Retrieve the journal entry.
	retrieved, err := repo.FindByID(ctx, entry.ID())
	require.NoError(t, err)

	// Verify all fields.
	assert.Equal(t, entry.ID(), retrieved.ID())
	assert.Equal(t, entry.TenantID(), retrieved.TenantID())
	assert.WithinDuration(t, entry.EffectiveDate(), retrieved.EffectiveDate(), time.Millisecond)
	assert.Equal(t, entry.Status(), retrieved.Status())
	assert.Equal(t, entry.Description(), retrieved.Description())
	assert.Equal(t, entry.Reference(), retrieved.Reference())
	assert.Equal(t, entry.Version(), retrieved.Version())

	// Verify postings.
	require.Len(t, retrieved.Postings(), 1)
	p := retrieved.Postings()[0]
	assert.Equal(t, "1000", p.DebitAccount().Code())
	assert.Equal(t, "2000", p.CreditAccount().Code())
	assert.True(t, decimal.NewFromInt(1000).Equal(p.Amount()))
	assert.Equal(t, "USD", p.Currency())
	assert.Equal(t, "Test posting", p.Description())
}

func TestJournalRepository_List(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewJournalRepo(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	baseTime := time.Now().UTC().Truncate(time.Microsecond)

	// Save 5 journal entries.
	for i := 0; i < 5; i++ {
		debit := valueobject.MustAccountCode("1000")
		credit := valueobject.MustAccountCode("2000")
		amount := decimal.NewFromInt(int64(100 * (i + 1)))
		posting, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "Posting")
		require.NoError(t, err)

		entry, err := model.NewJournalEntry(
			tenantID,
			baseTime.Add(time.Duration(i)*time.Hour),
			[]valueobject.PostingPair{posting},
			"Entry",
			"REF",
		)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, entry))
	}

	// List with pagination: page 1, limit 3.
	from := baseTime.Add(-time.Hour)
	to := baseTime.Add(10 * time.Hour)
	entries, total, err := repo.ListByTenant(ctx, tenantID, from, to, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, entries, 3)

	// List page 2.
	entries2, total2, err := repo.ListByTenant(ctx, tenantID, from, to, 3, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, total2)
	assert.Len(t, entries2, 2)

	// Entries from different pages should not overlap.
	ids := make(map[uuid.UUID]bool)
	for _, e := range entries {
		ids[e.ID()] = true
	}
	for _, e := range entries2 {
		assert.False(t, ids[e.ID()], "entry %s appears on both pages", e.ID())
	}
}

func TestBalanceRepository_GetBalance(t *testing.T) {
	pool := setupTestDB(t)
	balanceRepo := postgres.NewBalanceRepo(pool)
	ctx := context.Background()

	acctCode := valueobject.MustAccountCode("1000")

	// Initially balance should be zero.
	balance, err := balanceRepo.GetBalance(ctx, acctCode, "USD", time.Now())
	require.NoError(t, err)
	assert.True(t, decimal.Zero.Equal(balance), "expected zero balance, got %s", balance)

	// Update balance with a positive delta.
	err = balanceRepo.UpdateBalance(ctx, acctCode, "USD", decimal.NewFromInt(500))
	require.NoError(t, err)

	balance, err = balanceRepo.GetBalance(ctx, acctCode, "USD", time.Now())
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(500).Equal(balance), "expected 500, got %s", balance)

	// Update balance with another delta.
	err = balanceRepo.UpdateBalance(ctx, acctCode, "USD", decimal.NewFromInt(300))
	require.NoError(t, err)

	balance, err = balanceRepo.GetBalance(ctx, acctCode, "USD", time.Now())
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(800).Equal(balance), "expected 800, got %s", balance)

	// Update balance with a negative delta.
	err = balanceRepo.UpdateBalance(ctx, acctCode, "USD", decimal.NewFromInt(-200))
	require.NoError(t, err)

	balance, err = balanceRepo.GetBalance(ctx, acctCode, "USD", time.Now())
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(600).Equal(balance), "expected 600, got %s", balance)

	// Verify different currency has separate balance.
	balance, err = balanceRepo.GetBalance(ctx, acctCode, "EUR", time.Now())
	require.NoError(t, err)
	assert.True(t, decimal.Zero.Equal(balance), "expected zero EUR balance, got %s", balance)
}

func TestOutbox_EventsPersisted(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewJournalRepo(pool)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create a journal entry and post it (which generates a domain event).
	entry := newTestEntry(t, tenantID)
	posted, err := entry.Post(time.Now().UTC())
	require.NoError(t, err)

	// Save the posted entry; this should write domain events to the outbox.
	err = repo.Save(ctx, posted)
	require.NoError(t, err)

	// Query outbox table directly to verify events were persisted.
	var count int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox WHERE aggregate_id = $1",
		posted.ID(),
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "expected 1 outbox event for the posted entry")

	// Verify outbox event details.
	var aggregateType, eventType string
	err = pool.QueryRow(ctx,
		"SELECT aggregate_type, event_type FROM outbox WHERE aggregate_id = $1",
		posted.ID(),
	).Scan(&aggregateType, &eventType)
	require.NoError(t, err)
	assert.Equal(t, "JournalEntry", aggregateType)
	assert.Equal(t, "entry.posted", eventType)
}
