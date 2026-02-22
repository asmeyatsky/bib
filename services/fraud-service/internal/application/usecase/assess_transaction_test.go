package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
)

// --- Mock implementations ---

type mockAssessmentRepository struct {
	savedAssessment *model.TransactionAssessment
	saveFunc        func(ctx context.Context, assessment *model.TransactionAssessment) error
	findByIDFunc    func(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error)
}

func (m *mockAssessmentRepository) Save(ctx context.Context, assessment *model.TransactionAssessment) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, assessment)
	}
	m.savedAssessment = assessment
	return nil
}

func (m *mockAssessmentRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, tenantID, id)
	}
	return nil, fmt.Errorf("assessment not found")
}

func (m *mockAssessmentRepository) FindByTransactionID(_ context.Context, _, _ uuid.UUID) (*model.TransactionAssessment, error) {
	return nil, nil
}

func (m *mockAssessmentRepository) FindByAccountID(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*model.TransactionAssessment, error) {
	return nil, nil
}

type mockFraudEventPublisher struct {
	publishedEvents []interface{}
	publishFunc     func(ctx context.Context, events ...interface{}) error
}

func (m *mockFraudEventPublisher) Publish(ctx context.Context, evts ...interface{}) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

// --- Tests ---

func validAssessRequest() dto.AssessTransactionRequest {
	return dto.AssessTransactionRequest{
		TenantID:        uuid.New(),
		TransactionID:   uuid.New(),
		AccountID:       uuid.New(),
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		TransactionType: "transfer",
	}
}

func TestAssessTransaction_Execute(t *testing.T) {
	t.Run("successfully assesses a low-risk transaction", func(t *testing.T) {
		repo := &mockAssessmentRepository{}
		publisher := &mockFraudEventPublisher{}
		scorer := service.NewRiskScorer()

		uc := usecase.NewAssessTransaction(repo, publisher, scorer)

		req := validAssessRequest()
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.ID)
		assert.Equal(t, req.TenantID, resp.TenantID)
		assert.Equal(t, req.TransactionID, resp.TransactionID)
		assert.Equal(t, "LOW", resp.RiskLevel)
		assert.Equal(t, "APPROVE", resp.Decision)
		assert.NotNil(t, repo.savedAssessment)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("assesses a high-value transaction with elevated risk", func(t *testing.T) {
		repo := &mockAssessmentRepository{}
		publisher := &mockFraudEventPublisher{}
		scorer := service.NewRiskScorer()

		uc := usecase.NewAssessTransaction(repo, publisher, scorer)

		req := validAssessRequest()
		req.Amount = decimal.NewFromInt(55000) // very high value
		req.TransactionType = "wire_transfer"
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Contains(t, []string{"MEDIUM", "HIGH", "CRITICAL"}, resp.RiskLevel)
		assert.NotEmpty(t, resp.RiskSignals)
	})

	t.Run("fails with invalid request data", func(t *testing.T) {
		repo := &mockAssessmentRepository{}
		publisher := &mockFraudEventPublisher{}
		scorer := service.NewRiskScorer()

		uc := usecase.NewAssessTransaction(repo, publisher, scorer)

		req := validAssessRequest()
		req.TransactionID = uuid.Nil // invalid
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create assessment")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := &mockAssessmentRepository{
			saveFunc: func(ctx context.Context, assessment *model.TransactionAssessment) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockFraudEventPublisher{}
		scorer := service.NewRiskScorer()

		uc := usecase.NewAssessTransaction(repo, publisher, scorer)

		req := validAssessRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save assessment")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		repo := &mockAssessmentRepository{}
		publisher := &mockFraudEventPublisher{
			publishFunc: func(ctx context.Context, evts ...interface{}) error {
				return fmt.Errorf("kafka unavailable")
			},
		}
		scorer := service.NewRiskScorer()

		uc := usecase.NewAssessTransaction(repo, publisher, scorer)

		req := validAssessRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish events")
	})
}
