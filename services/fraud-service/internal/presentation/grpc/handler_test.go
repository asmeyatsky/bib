package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
)

// --- Mock implementations ---

type mockAssessmentRepo struct {
	saveErr      error
	findByIDFunc func(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error)
}

func (m *mockAssessmentRepo) Save(_ context.Context, _ *model.TransactionAssessment) error {
	return m.saveErr
}

func (m *mockAssessmentRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, tenantID, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockAssessmentRepo) FindByTransactionID(_ context.Context, _, _ uuid.UUID) (*model.TransactionAssessment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockAssessmentRepo) FindByAccountID(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*model.TransactionAssessment, error) {
	return nil, nil
}

type mockEventPublisher struct {
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, _ ...interface{}) error {
	return m.publishErr
}

// --- Helpers ---

func contextWithClaims() context.Context {
	claims := &auth.Claims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Roles:    []string{auth.RoleAdmin},
	}
	return auth.ContextWithClaims(context.Background(), claims)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func buildTestHandler() *FraudServiceHandler {
	repo := &mockAssessmentRepo{}
	publisher := &mockEventPublisher{}
	scorer := service.NewRiskScorer()
	logger := testLogger()

	return NewFraudServiceHandler(
		usecase.NewAssessTransaction(repo, publisher, scorer),
		usecase.NewGetAssessment(repo),
		logger,
	)
}

func buildHandlerWithRepo(repo *mockAssessmentRepo) *FraudServiceHandler {
	publisher := &mockEventPublisher{}
	scorer := service.NewRiskScorer()
	logger := testLogger()

	return NewFraudServiceHandler(
		usecase.NewAssessTransaction(repo, publisher, scorer),
		usecase.NewGetAssessment(repo),
		logger,
	)
}

// --- Tests ---

func TestAssessTransaction(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AssessTransaction(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("empty transaction_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: "",
			AccountID:     uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid transaction_id")
	})

	t.Run("invalid transaction_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: "bad-uuid",
			AccountID:     uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid transaction_id")
	})

	t.Run("invalid account_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: uuid.New().String(),
			AccountID:     "bad-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid account_id")
	})

	t.Run("invalid amount returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: uuid.New().String(),
			AccountID:     uuid.New().String(),
			Amount: &MoneyMsg{
				Amount:   "not-a-number",
				Currency: "USD",
			},
			TransactionType: "TRANSFER",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("happy path returns assessment", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: uuid.New().String(),
			AccountID:     uuid.New().String(),
			Amount: &MoneyMsg{
				Amount:   "100.00",
				Currency: "USD",
			},
			TransactionType: "TRANSFER",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Assessment)
		assert.NotEmpty(t, resp.Assessment.ID)
		assert.NotEmpty(t, resp.Assessment.RiskLevel)
		assert.NotEmpty(t, resp.Assessment.Decision)
	})

	t.Run("save failure returns Internal", func(t *testing.T) {
		repo := &mockAssessmentRepo{saveErr: fmt.Errorf("db error")}
		h := buildHandlerWithRepo(repo)

		_, err := h.AssessTransaction(contextWithClaims(), &AssessTransactionRequest{
			TenantID:      uuid.New().String(),
			TransactionID: uuid.New().String(),
			AccountID:     uuid.New().String(),
			Amount: &MoneyMsg{
				Amount:   "50.00",
				Currency: "USD",
			},
			TransactionType: "PAYMENT",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestGetAssessment(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetAssessment(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetAssessment(contextWithClaims(), &GetAssessmentRequest{ID: "bad-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found returns Internal", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetAssessment(contextWithClaims(), &GetAssessmentRequest{
			ID: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("happy path returns assessment", func(t *testing.T) {
		repo := &mockAssessmentRepo{
			findByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*model.TransactionAssessment, error) {
				// Create a mock assessment via the domain model
				return createTestAssessment(), nil
			},
		}
		h := buildHandlerWithRepo(repo)

		assessmentID := uuid.New()
		resp, err := h.GetAssessment(contextWithClaims(), &GetAssessmentRequest{
			ID: assessmentID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Assessment)
		assert.NotEmpty(t, resp.Assessment.ID)
		assert.NotEmpty(t, resp.Assessment.Decision)
	})
}

func TestToTransactionAssessmentMsg(t *testing.T) {
	assessment := createTestAssessment()
	resp := dto.FromModel(assessment)

	assert.Equal(t, assessment.ID(), resp.ID)
	assert.Equal(t, assessment.TenantID(), resp.TenantID)
	assert.Equal(t, assessment.TransactionID(), resp.TransactionID)
	assert.Equal(t, assessment.AccountID(), resp.AccountID)
	assert.Equal(t, assessment.RiskLevel().String(), resp.RiskLevel)
	assert.Equal(t, assessment.Decision().String(), resp.Decision)
	assert.Equal(t, assessment.RiskScore(), resp.RiskScore)
}

func createTestAssessment() *model.TransactionAssessment {
	a, _ := model.NewTransactionAssessment(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		decimal.NewFromInt(100),
		"USD",
		"TRANSFER",
	)
	// Assess it to set risk level and decision
	_ = a.Assess(25, []string{"low_amount"})
	return a
}

// requireGRPCCode asserts that an error is a gRPC status error with the given code.
func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	assert.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}
