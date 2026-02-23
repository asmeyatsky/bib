package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

func activeProduct() model.DepositProduct {
	tier, _ := valueobject.NewInterestTier(decimal.Zero, decimal.NewFromInt(100000), 250)
	product, _ := model.NewDepositProduct(uuid.New(), "Savings", "USD", []valueobject.InterestTier{tier}, 0)
	return product
}

func termProduct() model.DepositProduct {
	tier, _ := valueobject.NewInterestTier(decimal.Zero, decimal.NewFromInt(100000), 350)
	product, _ := model.NewDepositProduct(uuid.New(), "Term Deposit 90", "USD", []valueobject.InterestTier{tier}, 90)
	return product
}

func TestOpenDepositPosition_Execute(t *testing.T) {
	t.Run("successfully opens a demand deposit position", func(t *testing.T) {
		product := activeProduct()
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{}
		publisher := &mockDepositEventPublisher{}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: product.ID(),
			Principal: decimal.NewFromInt(1000),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.ID)
		assert.Equal(t, req.TenantID, resp.TenantID)
		assert.Equal(t, req.AccountID, resp.AccountID)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.True(t, decimal.NewFromInt(1000).Equal(resp.Principal))
		assert.Equal(t, "USD", resp.Currency)
		assert.Nil(t, resp.MaturityDate) // demand deposit has no maturity

		require.NotNil(t, positionRepo.savedPosition)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("successfully opens a term deposit with maturity date", func(t *testing.T) {
		product := termProduct()
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{}
		publisher := &mockDepositEventPublisher{}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: product.ID(),
			Principal: decimal.NewFromInt(5000),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp.MaturityDate) // term deposit has maturity
	})

	t.Run("fails when product not found", func(t *testing.T) {
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return model.DepositProduct{}, fmt.Errorf("product not found")
			},
		}
		positionRepo := &mockDepositPositionRepository{}
		publisher := &mockDepositEventPublisher{}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: uuid.New(),
			Principal: decimal.NewFromInt(1000),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "product not found")
	})

	t.Run("fails when product is inactive", func(t *testing.T) {
		product := activeProduct()
		deactivated, _ := product.Deactivate(time.Now())
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return deactivated, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{}
		publisher := &mockDepositEventPublisher{}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: product.ID(),
			Principal: decimal.NewFromInt(1000),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("fails when position save fails", func(t *testing.T) {
		product := activeProduct()
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{
			saveFunc: func(_ context.Context, _ model.DepositPosition) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockDepositEventPublisher{}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: product.ID(),
			Principal: decimal.NewFromInt(1000),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save deposit position")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		product := activeProduct()
		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{}
		publisher := &mockDepositEventPublisher{
			publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}

		uc := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)

		req := dto.OpenPositionRequest{
			TenantID:  uuid.New(),
			AccountID: uuid.New(),
			ProductID: product.ID(),
			Principal: decimal.NewFromInt(1000),
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish events")
	})
}
