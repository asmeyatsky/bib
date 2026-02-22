package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

type listMockJournalRepository struct {
	listByAccountFunc func(ctx context.Context, tenantID uuid.UUID, account valueobject.AccountCode, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error)
	listByTenantFunc  func(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error)
}

func (m *listMockJournalRepository) Save(_ context.Context, _ model.JournalEntry) error {
	return nil
}

func (m *listMockJournalRepository) FindByID(_ context.Context, _ uuid.UUID) (model.JournalEntry, error) {
	return model.JournalEntry{}, fmt.Errorf("not implemented")
}

func (m *listMockJournalRepository) ListByAccount(ctx context.Context, tenantID uuid.UUID, account valueobject.AccountCode, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
	if m.listByAccountFunc != nil {
		return m.listByAccountFunc(ctx, tenantID, account, from, to, limit, offset)
	}
	return nil, 0, nil
}

func (m *listMockJournalRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
	if m.listByTenantFunc != nil {
		return m.listByTenantFunc(ctx, tenantID, from, to, limit, offset)
	}
	return nil, 0, nil
}

func TestListJournalEntries_Execute(t *testing.T) {
	t.Run("lists entries by tenant", func(t *testing.T) {
		tenantID := uuid.New()
		entry := sampleJournalEntry()

		repo := &listMockJournalRepository{
			listByTenantFunc: func(ctx context.Context, tid uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
				assert.Equal(t, tenantID, tid)
				return []model.JournalEntry{entry}, 1, nil
			},
		}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{
			TenantID: tenantID,
			PageSize: 50,
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Entries, 1)
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("lists entries by account code", func(t *testing.T) {
		tenantID := uuid.New()
		entry := sampleJournalEntry()

		repo := &listMockJournalRepository{
			listByAccountFunc: func(ctx context.Context, tid uuid.UUID, account valueobject.AccountCode, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
				assert.Equal(t, "1000", account.Code())
				return []model.JournalEntry{entry}, 1, nil
			},
		}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{
			TenantID:    tenantID,
			AccountCode: "1000",
			PageSize:    50,
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Entries, 1)
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("applies default page size", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockJournalRepository{
			listByTenantFunc: func(ctx context.Context, tid uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
				assert.Equal(t, 50, limit)
				return nil, 0, nil
			},
		}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{TenantID: tenantID}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("caps page size at 1000", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockJournalRepository{
			listByTenantFunc: func(ctx context.Context, tid uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
				assert.Equal(t, 1000, limit)
				return nil, 0, nil
			},
		}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{TenantID: tenantID, PageSize: 5000}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("fails with invalid account code", func(t *testing.T) {
		repo := &listMockJournalRepository{}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{
			TenantID:    uuid.New(),
			AccountCode: "INVALID",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account code")
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := &listMockJournalRepository{
			listByTenantFunc: func(ctx context.Context, tid uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error) {
				return nil, 0, fmt.Errorf("database unavailable")
			},
		}

		uc := usecase.NewListJournalEntries(repo)

		req := dto.ListEntriesRequest{TenantID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list entries")
	})
}
