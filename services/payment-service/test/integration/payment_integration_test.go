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
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
	"github.com/bibbank/bib/services/payment-service/internal/infrastructure/postgres"
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

func newTestPaymentOrder(t *testing.T, tenantID uuid.UUID) model.PaymentOrder {
	t.Helper()
	sourceAcct := uuid.New()
	destAcct := uuid.New()
	amount := decimal.NewFromInt(2500)
	rail := valueobject.RailACH
	routingInfo, err := valueobject.NewRoutingInfo("021000021", "123456789")
	require.NoError(t, err)

	order, err := model.NewPaymentOrder(
		tenantID,
		sourceAcct,
		destAcct,
		amount,
		"USD",
		rail,
		routingInfo,
		"PAY-REF-001",
		"Test payment order",
	)
	require.NoError(t, err)

	// Clear domain events so Save does not try to write them to outbox
	// (the domain event from NewPaymentOrder references the events package interface).
	_, order = order.ClearDomainEvents()
	return order
}

func TestPaymentRepository_SaveAndGet(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewPaymentOrderRepo(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	order := newTestPaymentOrder(t, tenantID)

	// Save the payment order.
	err := repo.Save(ctx, order)
	require.NoError(t, err)

	// Retrieve the payment order.
	retrieved, err := repo.FindByID(ctx, order.ID())
	require.NoError(t, err)

	// Verify all fields.
	assert.Equal(t, order.ID(), retrieved.ID())
	assert.Equal(t, order.TenantID(), retrieved.TenantID())
	assert.Equal(t, order.SourceAccountID(), retrieved.SourceAccountID())
	assert.Equal(t, order.DestinationAccountID(), retrieved.DestinationAccountID())
	assert.True(t, order.Amount().Equal(retrieved.Amount()))
	assert.Equal(t, order.Currency(), retrieved.Currency())
	assert.Equal(t, order.Rail().String(), retrieved.Rail().String())
	assert.Equal(t, order.Status().String(), retrieved.Status().String())
	assert.Equal(t, order.RoutingInfo().RoutingNumber(), retrieved.RoutingInfo().RoutingNumber())
	assert.Equal(t, order.RoutingInfo().ExternalAccountNumber(), retrieved.RoutingInfo().ExternalAccountNumber())
	assert.Equal(t, order.Reference(), retrieved.Reference())
	assert.Equal(t, order.Description(), retrieved.Description())
	assert.Equal(t, order.Version(), retrieved.Version())
}

func TestPaymentRepository_List(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewPaymentOrderRepo(pool)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create 3 orders for tenant A and 2 for tenant B.
	for i := 0; i < 3; i++ {
		order := newTestPaymentOrder(t, tenantA)
		require.NoError(t, repo.Save(ctx, order))
	}
	for i := 0; i < 2; i++ {
		order := newTestPaymentOrder(t, tenantB)
		require.NoError(t, repo.Save(ctx, order))
	}

	// List tenant A orders.
	ordersA, totalA, err := repo.ListByTenant(ctx, tenantA, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, totalA)
	assert.Len(t, ordersA, 3)

	// All returned orders should belong to tenant A.
	for _, o := range ordersA {
		assert.Equal(t, tenantA, o.TenantID())
	}

	// List tenant B orders.
	ordersB, totalB, err := repo.ListByTenant(ctx, tenantB, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, totalB)
	assert.Len(t, ordersB, 2)

	for _, o := range ordersB {
		assert.Equal(t, tenantB, o.TenantID())
	}
}

func TestPaymentRepository_UpdateStatus(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewPaymentOrderRepo(pool)
	ctx := context.Background()

	tenantID := uuid.New()
	order := newTestPaymentOrder(t, tenantID)

	// Save initial order (version 1, status INITIATED).
	err := repo.Save(ctx, order)
	require.NoError(t, err)

	// Transition to PROCESSING.
	processing, err := order.MarkProcessing(time.Now().UTC())
	require.NoError(t, err)

	// Clear events before saving to avoid outbox insert issues.
	_, processing = processing.ClearDomainEvents()
	err = repo.Save(ctx, processing)
	require.NoError(t, err)

	// Retrieve and verify status and version.
	retrieved, err := repo.FindByID(ctx, order.ID())
	require.NoError(t, err)
	assert.Equal(t, "PROCESSING", retrieved.Status().String())
	assert.Equal(t, 2, retrieved.Version(), "version should be incremented after status update")

	// Transition to SETTLED.
	settled, err := processing.Settle(time.Now().UTC())
	require.NoError(t, err)
	_, settled = settled.ClearDomainEvents()
	err = repo.Save(ctx, settled)
	require.NoError(t, err)

	retrieved, err = repo.FindByID(ctx, order.ID())
	require.NoError(t, err)
	assert.Equal(t, "SETTLED", retrieved.Status().String())
	assert.Equal(t, 3, retrieved.Version(), "version should be 3 after second status update")
	assert.NotNil(t, retrieved.SettledAt(), "settled_at should be set")
}
