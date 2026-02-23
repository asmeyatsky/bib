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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/testutil"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
	"github.com/bibbank/bib/services/account-service/internal/infrastructure/postgres"
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

	return pg.Pool
}

func newTestAccount(t *testing.T, tenantID uuid.UUID) model.CustomerAccount {
	t.Helper()

	holder, err := model.NewAccountHolder(
		uuid.New(),
		"Jane",
		"Doe",
		"jane.doe@example.com",
		uuid.Nil,
	)
	require.NoError(t, err)

	account, err := model.NewCustomerAccount(
		tenantID,
		valueobject.AccountTypeChecking,
		"USD",
		holder,
	)
	require.NoError(t, err)

	// Clear domain events so Save does not write them to outbox
	// (the outbox INSERT in the repo expects specific event fields).
	account = account.ClearDomainEvents()
	return account
}

func TestAccountRepository_SaveAndGet(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewAccountRepository(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	account := newTestAccount(t, tenantID)

	// Save the account.
	err := repo.Save(ctx, account)
	require.NoError(t, err)

	// Retrieve the account.
	retrieved, err := repo.FindByID(ctx, account.ID())
	require.NoError(t, err)

	// Verify all fields.
	assert.Equal(t, account.ID(), retrieved.ID())
	assert.Equal(t, account.TenantID(), retrieved.TenantID())
	assert.Equal(t, account.AccountNumber().String(), retrieved.AccountNumber().String())
	assert.Equal(t, account.AccountType().String(), retrieved.AccountType().String())
	assert.Equal(t, account.Status(), retrieved.Status())
	assert.Equal(t, account.Currency(), retrieved.Currency())
	assert.Equal(t, account.Version(), retrieved.Version())

	// Verify holder.
	assert.Equal(t, account.Holder().ID(), retrieved.Holder().ID())
	assert.Equal(t, account.Holder().FirstName(), retrieved.Holder().FirstName())
	assert.Equal(t, account.Holder().LastName(), retrieved.Holder().LastName())
	assert.Equal(t, account.Holder().Email(), retrieved.Holder().Email())
}

func TestAccountRepository_ListByTenant(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewAccountRepository(pool)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create 3 accounts for tenant A and 2 for tenant B.
	for i := 0; i < 3; i++ {
		account := newTestAccount(t, tenantA)
		require.NoError(t, repo.Save(ctx, account))
	}
	for i := 0; i < 2; i++ {
		account := newTestAccount(t, tenantB)
		require.NoError(t, repo.Save(ctx, account))
	}

	// List tenant A accounts.
	accountsA, totalA, err := repo.ListByTenant(ctx, tenantA, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, totalA)
	assert.Len(t, accountsA, 3)

	for _, a := range accountsA {
		assert.Equal(t, tenantA, a.TenantID())
	}

	// List tenant B accounts.
	accountsB, totalB, err := repo.ListByTenant(ctx, tenantB, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, totalB)
	assert.Len(t, accountsB, 2)

	for _, a := range accountsB {
		assert.Equal(t, tenantB, a.TenantID())
	}
}

func TestAccountRepository_OptimisticLocking(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewAccountRepository(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	account := newTestAccount(t, tenantID)

	// Save the initial account (version 1).
	err := repo.Save(ctx, account)
	require.NoError(t, err)

	// Activate the account (version 2).
	activated, err := account.Activate(time.Now().UTC())
	require.NoError(t, err)
	activated = activated.ClearDomainEvents()
	err = repo.Save(ctx, activated)
	require.NoError(t, err)

	// Verify version was incremented.
	retrieved, err := repo.FindByID(ctx, account.ID())
	require.NoError(t, err)
	assert.Equal(t, 2, retrieved.Version())

	// Now try to save the stale version 1 account again (simulating a concurrent modification).
	// This should fail because the WHERE clause checks version = EXCLUDED.version - 1,
	// meaning it expects version 1 in the DB but it's now 2.
	staleUpdate, err := account.Activate(time.Now().UTC())
	require.NoError(t, err)
	staleUpdate = staleUpdate.ClearDomainEvents()
	err = repo.Save(ctx, staleUpdate)
	require.Error(t, err, "saving a stale version should fail with optimistic concurrency conflict")
	assert.Contains(t, err.Error(), "optimistic concurrency conflict")
}
