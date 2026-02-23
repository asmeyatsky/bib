package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

type listMockAccountRepository struct {
	listByTenantFunc func(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error)
	listByHolderFunc func(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error)
}

func (m *listMockAccountRepository) Save(_ context.Context, _ model.CustomerAccount) error {
	return nil
}

func (m *listMockAccountRepository) FindByID(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
	return model.CustomerAccount{}, fmt.Errorf("not implemented")
}

func (m *listMockAccountRepository) FindByAccountNumber(_ context.Context, _ valueobject.AccountNumber) (model.CustomerAccount, error) {
	return model.CustomerAccount{}, fmt.Errorf("not implemented")
}

func (m *listMockAccountRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	if m.listByTenantFunc != nil {
		return m.listByTenantFunc(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

func (m *listMockAccountRepository) ListByHolder(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	if m.listByHolderFunc != nil {
		return m.listByHolderFunc(ctx, holderID, limit, offset)
	}
	return nil, 0, nil
}

func sampleAccounts(tenantID uuid.UUID, count int) []model.CustomerAccount {
	var accounts []model.CustomerAccount
	for i := 0; i < count; i++ {
		holder := model.ReconstructAccountHolder(uuid.New(), "User", fmt.Sprintf("Test%d", i), fmt.Sprintf("user%d@example.com", i), uuid.New())
		acctType, _ := valueobject.NewAccountType("CHECKING")
		now := time.Now()
		accounts = append(accounts, model.ReconstructCustomerAccount(
			uuid.New(), tenantID, valueobject.NewAccountNumber(), acctType,
			model.AccountStatusActive, "USD", holder, fmt.Sprintf("2000-%03d", i), 1, now, now,
		))
	}
	return accounts
}

func TestListAccountsUseCase_Execute(t *testing.T) {
	t.Run("successfully lists accounts by tenant", func(t *testing.T) {
		tenantID := uuid.New()
		accounts := sampleAccounts(tenantID, 3)

		repo := &listMockAccountRepository{
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				assert.Equal(t, tenantID, tid)
				return accounts, 3, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{TenantID: tenantID, Limit: 20, Offset: 0}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Accounts, 3)
		assert.Equal(t, 3, resp.TotalCount)
	})

	t.Run("successfully lists accounts by holder", func(t *testing.T) {
		holderID := uuid.New()
		tenantID := uuid.New()
		accounts := sampleAccounts(tenantID, 2)

		repo := &listMockAccountRepository{
			listByHolderFunc: func(_ context.Context, hid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				assert.Equal(t, holderID, hid)
				return accounts, 2, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{HolderID: holderID, Limit: 20, Offset: 0}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Accounts, 2)
		assert.Equal(t, 2, resp.TotalCount)
	})

	t.Run("fails when neither tenant nor holder is provided", func(t *testing.T) {
		repo := &listMockAccountRepository{}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id or holder_id is required")
	})

	t.Run("applies default limit when not provided", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockAccountRepository{
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				assert.Equal(t, 20, limit) // default limit
				assert.Equal(t, 0, offset)
				return nil, 0, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{TenantID: tenantID}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("caps limit at 100", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockAccountRepository{
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				assert.Equal(t, 100, limit) // capped at max
				return nil, 0, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{TenantID: tenantID, Limit: 500}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockAccountRepository{
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				return nil, 0, fmt.Errorf("database unavailable")
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{TenantID: tenantID}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list accounts")
	})

	t.Run("holder takes priority over tenant", func(t *testing.T) {
		holderID := uuid.New()
		tenantID := uuid.New()
		holderCalled := false

		repo := &listMockAccountRepository{
			listByHolderFunc: func(_ context.Context, hid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				holderCalled = true
				return nil, 0, nil
			},
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
				t.Fatal("should not call ListByTenant when HolderID is set")
				return nil, 0, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewListAccountsUseCase(repo, logger)

		req := dto.ListAccountsRequest{TenantID: tenantID, HolderID: holderID}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.True(t, holderCalled)
	})
}
