package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
)

// --- Mock implementations ---

type mockDepositProductRepository struct {
	savedProduct *model.DepositProduct
	saveFunc     func(ctx context.Context, product model.DepositProduct) error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.DepositProduct, error)
}

func (m *mockDepositProductRepository) Save(ctx context.Context, product model.DepositProduct) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, product)
	}
	m.savedProduct = &product
	return nil
}

func (m *mockDepositProductRepository) FindByID(ctx context.Context, id uuid.UUID) (model.DepositProduct, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.DepositProduct{}, fmt.Errorf("product not found: %s", id)
}

func (m *mockDepositProductRepository) ListByTenant(_ context.Context, _ uuid.UUID) ([]model.DepositProduct, error) {
	return nil, nil
}

type mockDepositPositionRepository struct {
	savedPosition *model.DepositPosition
	saveFunc      func(ctx context.Context, position model.DepositPosition) error
	findByIDFunc  func(ctx context.Context, id uuid.UUID) (model.DepositPosition, error)
	findActiveFunc func(ctx context.Context, tenantID uuid.UUID) ([]model.DepositPosition, error)
}

func (m *mockDepositPositionRepository) Save(ctx context.Context, position model.DepositPosition) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, position)
	}
	m.savedPosition = &position
	return nil
}

func (m *mockDepositPositionRepository) FindByID(ctx context.Context, id uuid.UUID) (model.DepositPosition, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.DepositPosition{}, fmt.Errorf("position not found: %s", id)
}

func (m *mockDepositPositionRepository) FindActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.DepositPosition, error) {
	if m.findActiveFunc != nil {
		return m.findActiveFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDepositPositionRepository) FindByAccount(_ context.Context, _ uuid.UUID) ([]model.DepositPosition, error) {
	return nil, nil
}

type mockDepositEventPublisher struct {
	publishedEvents []events.DomainEvent
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
}

func (m *mockDepositEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

// --- Tests ---

func validCreateProductRequest() dto.CreateDepositProductRequest {
	return dto.CreateDepositProductRequest{
		TenantID: uuid.New(),
		Name:     "Premium Savings",
		Currency: "USD",
		Tiers: []dto.InterestTierDTO{
			{
				MinBalance: decimal.Zero,
				MaxBalance: decimal.NewFromInt(10000),
				RateBps:    150,
			},
		},
		TermDays: 0,
	}
}

func TestCreateDepositProduct_Execute(t *testing.T) {
	t.Run("successfully creates a demand deposit product", func(t *testing.T) {
		repo := &mockDepositProductRepository{}
		uc := usecase.NewCreateDepositProduct(repo)

		req := validCreateProductRequest()
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.ID)
		assert.Equal(t, req.TenantID, resp.TenantID)
		assert.Equal(t, "Premium Savings", resp.Name)
		assert.Equal(t, "USD", resp.Currency)
		assert.True(t, resp.IsActive)
		assert.Equal(t, 0, resp.TermDays)
		assert.Len(t, resp.Tiers, 1)

		require.NotNil(t, repo.savedProduct)
	})

	t.Run("successfully creates a term deposit product", func(t *testing.T) {
		repo := &mockDepositProductRepository{}
		uc := usecase.NewCreateDepositProduct(repo)

		req := validCreateProductRequest()
		req.TermDays = 90
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 90, resp.TermDays)
	})

	t.Run("fails with invalid tier", func(t *testing.T) {
		repo := &mockDepositProductRepository{}
		uc := usecase.NewCreateDepositProduct(repo)

		req := validCreateProductRequest()
		req.Tiers = []dto.InterestTierDTO{
			{
				MinBalance: decimal.NewFromInt(100),
				MaxBalance: decimal.NewFromInt(50), // max < min
				RateBps:    150,
			},
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid interest tier")
	})

	t.Run("fails when product validation fails", func(t *testing.T) {
		repo := &mockDepositProductRepository{}
		uc := usecase.NewCreateDepositProduct(repo)

		req := validCreateProductRequest()
		req.Name = "" // empty name
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create deposit product")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := &mockDepositProductRepository{
			saveFunc: func(ctx context.Context, product model.DepositProduct) error {
				return fmt.Errorf("database unavailable")
			},
		}
		uc := usecase.NewCreateDepositProduct(repo)

		req := validCreateProductRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save deposit product")
	})
}
