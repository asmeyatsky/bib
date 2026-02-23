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
	"github.com/bibbank/bib/services/deposit-service/internal/domain/service"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

func TestAccrueInterest_Execute(t *testing.T) {
	t.Run("successfully accrues interest for active positions", func(t *testing.T) {
		tenantID := uuid.New()
		productID := uuid.New()

		// Create position from 30 days ago
		yesterday := time.Now().UTC().AddDate(0, 0, -30)
		position := model.ReconstructPosition(
			uuid.New(), tenantID, uuid.New(), productID,
			decimal.NewFromInt(10000), "USD",
			decimal.Zero, model.PositionStatusActive,
			yesterday, nil, yesterday, 1,
			yesterday, yesterday,
		)

		tier, _ := valueobject.NewInterestTier(decimal.Zero, decimal.NewFromInt(100000), 250)
		product := model.ReconstructProduct(
			productID, tenantID, "Savings", "USD",
			[]valueobject.InterestTier{tier}, 0, true, 1,
			yesterday, yesterday,
		)

		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}

		var savedPositions []model.DepositPosition
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, tid uuid.UUID) ([]model.DepositPosition, error) {
				assert.Equal(t, tenantID, tid)
				return []model.DepositPosition{position}, nil
			},
			saveFunc: func(_ context.Context, pos model.DepositPosition) error {
				savedPositions = append(savedPositions, pos)
				return nil
			},
		}
		publisher := &mockDepositEventPublisher{}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{
			TenantID: tenantID,
			AsOf:     time.Now().UTC(),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 1, resp.PositionsProcessed)
		assert.True(t, resp.TotalAccrued.GreaterThan(decimal.Zero))
		assert.Len(t, savedPositions, 1)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("handles no active positions", func(t *testing.T) {
		tenantID := uuid.New()

		productRepo := &mockDepositProductRepository{}
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
				return nil, nil
			},
		}
		publisher := &mockDepositEventPublisher{}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{TenantID: tenantID, AsOf: time.Now().UTC()}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 0, resp.PositionsProcessed)
		assert.True(t, resp.TotalAccrued.Equal(decimal.Zero))
	})

	t.Run("fails when fetching active positions fails", func(t *testing.T) {
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
				return nil, fmt.Errorf("database unavailable")
			},
		}
		productRepo := &mockDepositProductRepository{}
		publisher := &mockDepositEventPublisher{}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{TenantID: uuid.New(), AsOf: time.Now().UTC()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch active positions")
	})

	t.Run("fails when product not found for position", func(t *testing.T) {
		tenantID := uuid.New()
		productID := uuid.New()

		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		position := model.ReconstructPosition(
			uuid.New(), tenantID, uuid.New(), productID,
			decimal.NewFromInt(10000), "USD",
			decimal.Zero, model.PositionStatusActive,
			yesterday, nil, yesterday, 1,
			yesterday, yesterday,
		)

		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return model.DepositProduct{}, fmt.Errorf("product not found")
			},
		}
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
				return []model.DepositPosition{position}, nil
			},
		}
		publisher := &mockDepositEventPublisher{}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{TenantID: tenantID, AsOf: time.Now().UTC()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch product")
	})

	t.Run("fails when position save fails", func(t *testing.T) {
		tenantID := uuid.New()
		productID := uuid.New()

		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		position := model.ReconstructPosition(
			uuid.New(), tenantID, uuid.New(), productID,
			decimal.NewFromInt(10000), "USD",
			decimal.Zero, model.PositionStatusActive,
			yesterday, nil, yesterday, 1,
			yesterday, yesterday,
		)

		tier, _ := valueobject.NewInterestTier(decimal.Zero, decimal.NewFromInt(100000), 250)
		product := model.ReconstructProduct(
			productID, tenantID, "Savings", "USD",
			[]valueobject.InterestTier{tier}, 0, true, 1,
			yesterday, yesterday,
		)

		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
				return []model.DepositPosition{position}, nil
			},
			saveFunc: func(_ context.Context, _ model.DepositPosition) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockDepositEventPublisher{}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{TenantID: tenantID, AsOf: time.Now().UTC()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save position")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		tenantID := uuid.New()
		productID := uuid.New()

		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		position := model.ReconstructPosition(
			uuid.New(), tenantID, uuid.New(), productID,
			decimal.NewFromInt(10000), "USD",
			decimal.Zero, model.PositionStatusActive,
			yesterday, nil, yesterday, 1,
			yesterday, yesterday,
		)

		tier, _ := valueobject.NewInterestTier(decimal.Zero, decimal.NewFromInt(100000), 250)
		product := model.ReconstructProduct(
			productID, tenantID, "Savings", "USD",
			[]valueobject.InterestTier{tier}, 0, true, 1,
			yesterday, yesterday,
		)

		productRepo := &mockDepositProductRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositProduct, error) {
				return product, nil
			},
		}
		positionRepo := &mockDepositPositionRepository{
			findActiveFunc: func(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
				return []model.DepositPosition{position}, nil
			},
		}
		publisher := &mockDepositEventPublisher{
			publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}
		engine := service.NewAccrualEngine()

		uc := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, engine)

		req := dto.AccrueInterestRequest{TenantID: tenantID, AsOf: time.Now().UTC()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish events")
	})
}
