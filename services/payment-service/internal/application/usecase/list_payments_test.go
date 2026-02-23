package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
)

type listMockPaymentOrderRepository struct {
	listByAccountFunc func(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error)
	listByTenantFunc  func(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error)
}

func (m *listMockPaymentOrderRepository) Save(_ context.Context, _ model.PaymentOrder) error {
	return nil
}

func (m *listMockPaymentOrderRepository) FindByID(_ context.Context, _ uuid.UUID) (model.PaymentOrder, error) {
	return model.PaymentOrder{}, fmt.Errorf("not implemented")
}

func (m *listMockPaymentOrderRepository) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	if m.listByAccountFunc != nil {
		return m.listByAccountFunc(ctx, accountID, limit, offset)
	}
	return nil, 0, nil
}

func (m *listMockPaymentOrderRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	if m.listByTenantFunc != nil {
		return m.listByTenantFunc(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

func TestListPayments_Execute(t *testing.T) {
	t.Run("lists payments by tenant", func(t *testing.T) {
		tenantID := uuid.New()
		order := samplePaymentOrder()

		repo := &listMockPaymentOrderRepository{
			listByTenantFunc: func(_ context.Context, tid uuid.UUID, _ int, _ int) ([]model.PaymentOrder, int, error) {
				assert.Equal(t, tenantID, tid)
				return []model.PaymentOrder{order}, 1, nil
			},
		}

		uc := usecase.NewListPayments(repo)

		req := dto.ListPaymentsRequest{TenantID: tenantID, PageSize: 20}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Payments, 1)
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("lists payments by account", func(t *testing.T) {
		accountID := uuid.New()
		order := samplePaymentOrder()

		repo := &listMockPaymentOrderRepository{
			listByAccountFunc: func(_ context.Context, aid uuid.UUID, _ int, _ int) ([]model.PaymentOrder, int, error) {
				assert.Equal(t, accountID, aid)
				return []model.PaymentOrder{order}, 1, nil
			},
		}

		uc := usecase.NewListPayments(repo)

		req := dto.ListPaymentsRequest{
			TenantID:  uuid.New(),
			AccountID: accountID,
			PageSize:  20,
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Payments, 1)
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("applies default page size", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockPaymentOrderRepository{
			listByTenantFunc: func(_ context.Context, _ uuid.UUID, limit, _ int) ([]model.PaymentOrder, int, error) {
				assert.Equal(t, 20, limit) // default
				return nil, 0, nil
			},
		}

		uc := usecase.NewListPayments(repo)

		req := dto.ListPaymentsRequest{TenantID: tenantID}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		tenantID := uuid.New()

		repo := &listMockPaymentOrderRepository{
			listByTenantFunc: func(_ context.Context, _ uuid.UUID, limit, _ int) ([]model.PaymentOrder, int, error) {
				assert.Equal(t, 100, limit) // capped
				return nil, 0, nil
			},
		}

		uc := usecase.NewListPayments(repo)

		req := dto.ListPaymentsRequest{TenantID: tenantID, PageSize: 500}
		_, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := &listMockPaymentOrderRepository{
			listByTenantFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]model.PaymentOrder, int, error) {
				return nil, 0, fmt.Errorf("database unavailable")
			},
		}

		uc := usecase.NewListPayments(repo)

		req := dto.ListPaymentsRequest{TenantID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list payment orders")
	})
}
