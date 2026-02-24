package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
	"github.com/bibbank/bib/services/account-service/internal/domain/event"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockAccountRepo struct {
	savedAccount *model.CustomerAccount
	saveErr      error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error)
	listFunc     func(ctx context.Context, id uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error)
}

func (m *mockAccountRepo) Save(_ context.Context, account model.CustomerAccount) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedAccount = &account
	return nil
}

func (m *mockAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.CustomerAccount{}, fmt.Errorf("account not found")
}

func (m *mockAccountRepo) FindByAccountNumber(_ context.Context, _ valueobject.AccountNumber) (model.CustomerAccount, error) {
	return model.CustomerAccount{}, fmt.Errorf("not implemented")
}

func (m *mockAccountRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockAccountRepo) ListByHolder(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, holderID, limit, offset)
	}
	return nil, 0, nil
}

type mockEventPublisher struct {
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, _ string, _ ...event.DomainEvent) error {
	return m.publishErr
}

type mockLedgerClient struct {
	createErr error
}

func (m *mockLedgerClient) CreateLedgerAccount(_ context.Context, _ uuid.UUID, _ string, _ string) error {
	return m.createErr
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

func buildTestHandler() (*AccountHandler, *mockAccountRepo) {
	repo := &mockAccountRepo{}
	publisher := &mockEventPublisher{}
	ledger := &mockLedgerClient{}
	logger := testLogger()

	return NewAccountHandler(
		usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger),
		usecase.NewGetAccountUseCase(repo, logger),
		usecase.NewFreezeAccountUseCase(repo, publisher, logger),
		usecase.NewCloseAccountUseCase(repo, publisher, logger),
		usecase.NewListAccountsUseCase(repo, logger),
	), repo
}

func makeActiveAccount(tenantID uuid.UUID) model.CustomerAccount {
	an := valueobject.NewAccountNumber()
	at, _ := valueobject.NewAccountType("CHECKING")
	holder := model.ReconstructAccountHolder(uuid.New(), "Jane", "Smith", "jane@example.com", uuid.Nil)
	now := time.Now().UTC()

	return model.ReconstructCustomerAccount(
		uuid.New(), tenantID, an, at,
		model.AccountStatusActive, "USD", holder,
		"2000-100", 1, now, now,
	)
}

// --- Tests ---

func TestOpenAccount(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.OpenAccount(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("missing currency returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.OpenAccount(contextWithClaims(), &OpenAccountRequest{
			TenantID:        uuid.New().String(),
			AccountType:     "CHECKING",
			Currency:        "",
			HolderFirstName: "Jane",
			HolderLastName:  "Smith",
			HolderEmail:     "jane@example.com",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "currency is required")
	})

	t.Run("invalid identity_verification_id returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.OpenAccount(contextWithClaims(), &OpenAccountRequest{
			TenantID:               uuid.New().String(),
			AccountType:            "CHECKING",
			Currency:               "USD",
			HolderFirstName:        "Jane",
			HolderLastName:         "Smith",
			HolderEmail:            "jane@example.com",
			IdentityVerificationID: "bad-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid identity_verification_id")
	})

	t.Run("happy path returns account info", func(t *testing.T) {
		h, _ := buildTestHandler()
		resp, err := h.OpenAccount(contextWithClaims(), &OpenAccountRequest{
			TenantID:        uuid.New().String(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "Jane",
			HolderLastName:  "Smith",
			HolderEmail:     "jane@example.com",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccountID)
		assert.NotEmpty(t, resp.AccountNumber)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.NotEmpty(t, resp.LedgerAccountCode)
	})

	t.Run("use case error returns Internal", func(t *testing.T) {
		repo := &mockAccountRepo{saveErr: fmt.Errorf("db error")}
		publisher := &mockEventPublisher{}
		ledger := &mockLedgerClient{}
		logger := testLogger()

		h := NewAccountHandler(
			usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger),
			usecase.NewGetAccountUseCase(repo, logger),
			usecase.NewFreezeAccountUseCase(repo, publisher, logger),
			usecase.NewCloseAccountUseCase(repo, publisher, logger),
			usecase.NewListAccountsUseCase(repo, logger),
		)

		_, err := h.OpenAccount(contextWithClaims(), &OpenAccountRequest{
			TenantID:        uuid.New().String(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "Jane",
			HolderLastName:  "Smith",
			HolderEmail:     "jane@example.com",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestGetAccount(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.GetAccount(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.GetAccount(contextWithClaims(), &GetAccountRequest{ID: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns account", func(t *testing.T) {
		h, repo := buildTestHandler()
		tenantID := uuid.New()
		account := makeActiveAccount(tenantID)

		repo.findByIDFunc = func(_ context.Context, id uuid.UUID) (model.CustomerAccount, error) {
			if id == account.ID() {
				return account, nil
			}
			return model.CustomerAccount{}, fmt.Errorf("not found")
		}

		resp, err := h.GetAccount(contextWithClaims(), &GetAccountRequest{ID: account.ID().String()})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, account.ID().String(), resp.AccountID)
		assert.Equal(t, "ACTIVE", resp.Status)
	})

	t.Run("not found returns NotFound", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.GetAccount(contextWithClaims(), &GetAccountRequest{ID: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestFreezeAccount(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.FreezeAccount(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.FreezeAccount(contextWithClaims(), &FreezeAccountRequest{ID: "bad", Reason: "fraud"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns frozen account", func(t *testing.T) {
		h, repo := buildTestHandler()
		account := makeActiveAccount(uuid.New())

		repo.findByIDFunc = func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
			return account, nil
		}

		resp, err := h.FreezeAccount(contextWithClaims(), &FreezeAccountRequest{
			ID:     account.ID().String(),
			Reason: "suspected fraud",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "FROZEN", resp.Status)
	})
}

func TestCloseAccount(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.CloseAccount(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid id returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.CloseAccount(contextWithClaims(), &CloseAccountRequest{ID: "bad", Reason: "closing"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns closed account", func(t *testing.T) {
		h, repo := buildTestHandler()
		account := makeActiveAccount(uuid.New())

		repo.findByIDFunc = func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
			return account, nil
		}

		resp, err := h.CloseAccount(contextWithClaims(), &CloseAccountRequest{
			ID:     account.ID().String(),
			Reason: "customer request",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "CLOSED", resp.Status)
	})
}

func TestListAccounts(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.ListAccounts(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid page_size returns InvalidArgument", func(t *testing.T) {
		h, _ := buildTestHandler()
		_, err := h.ListAccounts(contextWithClaims(), &ListAccountsRequest{TenantID: uuid.New().String(), PageSize: -1})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns accounts list", func(t *testing.T) {
		h, repo := buildTestHandler()
		tenantID := uuid.New()
		account1 := makeActiveAccount(tenantID)
		account2 := makeActiveAccount(tenantID)

		repo.listFunc = func(_ context.Context, _ uuid.UUID, _, _ int) ([]model.CustomerAccount, int, error) {
			return []model.CustomerAccount{account1, account2}, 2, nil
		}

		resp, err := h.ListAccounts(contextWithClaims(), &ListAccountsRequest{
			TenantID: tenantID.String(),
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Len(t, resp.Accounts, 2)
		assert.Equal(t, int32(2), resp.TotalCount)
	})

	t.Run("empty holder_id ignored, uses tenant_id", func(t *testing.T) {
		h, repo := buildTestHandler()
		tenantID := uuid.New()

		repo.listFunc = func(_ context.Context, _ uuid.UUID, _, _ int) ([]model.CustomerAccount, int, error) {
			return nil, 0, nil
		}

		resp, err := h.ListAccounts(contextWithClaims(), &ListAccountsRequest{
			TenantID: tenantID.String(),
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Accounts)
	})
}

func TestToAccountMsg(t *testing.T) {
	accountID := uuid.New()
	tenantID := uuid.New()

	msg := toAccountMsg(dto.AccountResponse{
		AccountID:         accountID,
		TenantID:          tenantID,
		AccountNumber:     "BIB-AAAA-BBBB-CCCC",
		AccountType:       "CHECKING",
		Status:            "ACTIVE",
		Currency:          "USD",
		LedgerAccountCode: "2000-100",
		HolderFirstName:   "Jane",
		HolderLastName:    "Smith",
		HolderEmail:       "jane@example.com",
		Version:           3,
	})

	assert.Equal(t, accountID.String(), msg.AccountID)
	assert.Equal(t, tenantID.String(), msg.TenantID)
	assert.Equal(t, "BIB-AAAA-BBBB-CCCC", msg.AccountNumber)
	assert.Equal(t, "CHECKING", msg.AccountType)
	assert.Equal(t, "ACTIVE", msg.Status)
	assert.Equal(t, "USD", msg.Currency)
	assert.Equal(t, "2000-100", msg.LedgerAccountCode)
	assert.Equal(t, int32(3), msg.Version)
}

// requireGRPCCode asserts that an error is a gRPC status error with the given code.
func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	assert.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}
