package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
	"github.com/bibbank/bib/services/card-service/internal/domain/event"
	"github.com/bibbank/bib/services/card-service/internal/domain/model"
	"github.com/bibbank/bib/services/card-service/internal/domain/service"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockCardRepo struct {
	saveErr      error
	updateErr    error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.Card, error)
	saveTxnErr   error
}

func (m *mockCardRepo) Save(_ context.Context, _ model.Card) error {
	return m.saveErr
}

func (m *mockCardRepo) Update(_ context.Context, _ model.Card) error {
	return m.updateErr
}

func (m *mockCardRepo) FindByID(ctx context.Context, id uuid.UUID) (model.Card, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.Card{}, fmt.Errorf("card not found")
}

func (m *mockCardRepo) FindByAccountID(_ context.Context, _ uuid.UUID) ([]model.Card, error) {
	return nil, nil
}

func (m *mockCardRepo) FindByTenantID(_ context.Context, _ uuid.UUID) ([]model.Card, error) {
	return nil, nil
}

func (m *mockCardRepo) SaveTransaction(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _, _, _, _, _ string) error {
	return m.saveTxnErr
}

type mockEventPublisher struct {
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, _ []event.DomainEvent) error {
	return m.publishErr
}

type mockCardProcessor struct{}

func (m *mockCardProcessor) IssuePhysicalCard(_ context.Context, _ model.Card) error {
	return nil
}

func (m *mockCardProcessor) GetCardDetails(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockBalanceClient struct {
	balance    decimal.Decimal
	balanceErr error
}

func (m *mockBalanceClient) GetAvailableBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	if m.balanceErr != nil {
		return decimal.Zero, m.balanceErr
	}
	return m.balance, nil
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

func buildTestHandler() *CardServiceHandler {
	repo := &mockCardRepo{}
	publisher := &mockEventPublisher{}
	processor := &mockCardProcessor{}
	balanceClient := &mockBalanceClient{balance: decimal.NewFromInt(10000)}
	jitFunding := service.NewJITFundingService()

	return NewCardServiceHandler(
		usecase.NewIssueCardUseCase(repo, publisher, processor),
		usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding),
		usecase.NewGetCardUseCase(repo),
		usecase.NewFreezeCardUseCase(repo, publisher),
	)
}

func buildHandlerWithRepo(repo *mockCardRepo) *CardServiceHandler {
	publisher := &mockEventPublisher{}
	processor := &mockCardProcessor{}
	balanceClient := &mockBalanceClient{balance: decimal.NewFromInt(10000)}
	jitFunding := service.NewJITFundingService()

	return NewCardServiceHandler(
		usecase.NewIssueCardUseCase(repo, publisher, processor),
		usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding),
		usecase.NewGetCardUseCase(repo),
		usecase.NewFreezeCardUseCase(repo, publisher),
	)
}

func makeTestCard() model.Card {
	ct, _ := valueobject.NewCardType("VIRTUAL")
	cs, _ := valueobject.NewCardStatus("ACTIVE")
	cn, _ := valueobject.NewCardNumber("1234", "12", "2030")

	return model.Reconstruct(
		uuid.New(), uuid.New(), uuid.New(),
		ct, cs, cn,
		"USD", decimal.NewFromInt(5000), decimal.NewFromInt(20000),
		decimal.Zero, decimal.Zero,
		1, time.Now().UTC(), time.Now().UTC(),
	)
}

// --- Tests ---

func TestIssueCard(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.IssueCard(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("missing card_type returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.IssueCard(contextWithClaims(), &IssueCardRequest{
			TenantID:  uuid.New().String(),
			AccountID: uuid.New().String(),
			CardType:  "",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "card_type is required")
	})

	t.Run("invalid account_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.IssueCard(contextWithClaims(), &IssueCardRequest{
			TenantID:  uuid.New().String(),
			AccountID: "bad-uuid",
			CardType:  "VIRTUAL",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid account_id")
	})

	t.Run("invalid daily_limit amount returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.IssueCard(contextWithClaims(), &IssueCardRequest{
			TenantID:  uuid.New().String(),
			AccountID: uuid.New().String(),
			CardType:  "VIRTUAL",
			DailyLimit: &MoneyMsg{
				Amount:   "not-a-number",
				Currency: "USD",
			},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid daily_limit amount")
	})

	t.Run("happy path issues a card", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.IssueCard(contextWithClaims(), &IssueCardRequest{
			TenantID:  uuid.New().String(),
			AccountID: uuid.New().String(),
			CardType:  "VIRTUAL",
			DailyLimit: &MoneyMsg{
				Amount:   "5000",
				Currency: "USD",
			},
			MonthlyLimit: &MoneyMsg{
				Amount:   "20000",
				Currency: "USD",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Card)
		assert.NotEmpty(t, resp.Card.ID)
		assert.Equal(t, "VIRTUAL", resp.Card.CardType)
		assert.NotEmpty(t, resp.Card.LastFour)
	})

	t.Run("save failure returns Internal", func(t *testing.T) {
		repo := &mockCardRepo{saveErr: fmt.Errorf("db error")}
		h := buildHandlerWithRepo(repo)

		_, err := h.IssueCard(contextWithClaims(), &IssueCardRequest{
			TenantID:  uuid.New().String(),
			AccountID: uuid.New().String(),
			CardType:  "VIRTUAL",
			DailyLimit: &MoneyMsg{
				Amount:   "1000",
				Currency: "USD",
			},
			MonthlyLimit: &MoneyMsg{
				Amount:   "5000",
				Currency: "USD",
			},
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestAuthorizeTransaction(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AuthorizeTransaction(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid card_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AuthorizeTransaction(contextWithClaims(), &AuthorizeTransactionRequest{
			CardID: "bad-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid card_id")
	})

	t.Run("invalid amount returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.AuthorizeTransaction(contextWithClaims(), &AuthorizeTransactionRequest{
			CardID: uuid.New().String(),
			Amount: &MoneyMsg{Amount: "not-a-number", Currency: "USD"},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid amount")
	})
}

func TestGetCard(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetCard(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetCard(contextWithClaims(), &GetCardRequest{ID: "bad-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns card details", func(t *testing.T) {
		card := makeTestCard()
		repo := &mockCardRepo{
			findByIDFunc: func(_ context.Context, id uuid.UUID) (model.Card, error) {
				return card, nil
			},
		}
		h := buildHandlerWithRepo(repo)

		resp, err := h.GetCard(contextWithClaims(), &GetCardRequest{ID: card.ID().String()})
		require.NoError(t, err)
		require.NotNil(t, resp.Card)
		assert.Equal(t, card.ID().String(), resp.Card.ID)
		assert.Equal(t, "VIRTUAL", resp.Card.CardType)
		assert.Equal(t, "ACTIVE", resp.Card.Status)
	})

	t.Run("not found returns Internal", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.GetCard(contextWithClaims(), &GetCardRequest{ID: uuid.New().String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestFreezeCard(t *testing.T) {
	t.Run("invalid card_id returns error", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.FreezeCard(contextWithClaims(), "bad-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid card_id")
	})

	t.Run("happy path freezes card", func(t *testing.T) {
		card := makeTestCard()
		repo := &mockCardRepo{
			findByIDFunc: func(_ context.Context, id uuid.UUID) (model.Card, error) {
				return card, nil
			},
		}
		h := buildHandlerWithRepo(repo)

		resp, err := h.FreezeCard(contextWithClaims(), card.ID().String())
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "FROZEN", resp.Status)
	})
}

// requireGRPCCode asserts that an error is a gRPC status error with the given code.
func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	assert.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}
