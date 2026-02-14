package usecase_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
	"github.com/bibbank/bib/services/account-service/internal/domain/event"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockAccountRepository struct {
	savedAccount *model.CustomerAccount
	saveErr      error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error)
}

func (m *mockAccountRepository) Save(ctx context.Context, account model.CustomerAccount) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedAccount = &account
	return nil
}

func (m *mockAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.CustomerAccount{}, fmt.Errorf("account not found")
}

func (m *mockAccountRepository) FindByAccountNumber(ctx context.Context, number valueobject.AccountNumber) (model.CustomerAccount, error) {
	return model.CustomerAccount{}, fmt.Errorf("not implemented")
}

func (m *mockAccountRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (m *mockAccountRepository) ListByHolder(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

type mockEventPublisher struct {
	publishedEvents []event.DomainEvent
	publishedTopic  string
	publishErr      error
}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, events ...event.DomainEvent) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedTopic = topic
	m.publishedEvents = append(m.publishedEvents, events...)
	return nil
}

type mockLedgerClient struct {
	createCalled    bool
	createdCode     string
	createdCurrency string
	createErr       error
}

func (m *mockLedgerClient) CreateLedgerAccount(ctx context.Context, tenantID uuid.UUID, accountCode string, currency string) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createCalled = true
	m.createdCode = accountCode
	m.createdCurrency = currency
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Tests ---

func TestOpenAccountUseCase_Execute(t *testing.T) {
	t.Run("successfully opens a checking account", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		ledger := &mockLedgerClient{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger)

		req := dto.OpenAccountRequest{
			TenantID:               uuid.New(),
			AccountType:            "CHECKING",
			Currency:               "USD",
			HolderFirstName:        "Jane",
			HolderLastName:         "Smith",
			HolderEmail:            "jane.smith@example.com",
			IdentityVerificationID: uuid.New(),
		}

		resp, err := uc.Execute(context.Background(), req)
		require.NoError(t, err)

		// Verify response.
		assert.NotEqual(t, uuid.Nil, resp.AccountID)
		assert.NotEmpty(t, resp.AccountNumber)
		assert.Equal(t, "PENDING", resp.Status)
		assert.NotEmpty(t, resp.LedgerAccountCode)
		assert.Contains(t, resp.LedgerAccountCode, "2000-")
		assert.False(t, resp.CreatedAt.IsZero())

		// Verify repository was called.
		require.NotNil(t, repo.savedAccount)
		assert.Equal(t, req.TenantID, repo.savedAccount.TenantID())

		// Verify ledger client was called.
		assert.True(t, ledger.createCalled)
		assert.Equal(t, resp.LedgerAccountCode, ledger.createdCode)
		assert.Equal(t, "USD", ledger.createdCurrency)

		// Verify events were published.
		assert.Equal(t, "account-events", publisher.publishedTopic)
		require.Len(t, publisher.publishedEvents, 1)
		assert.Equal(t, "account.opened", publisher.publishedEvents[0].EventType())
	})

	t.Run("successfully opens a savings account", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		ledger := &mockLedgerClient{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "SAVINGS",
			Currency:        "EUR",
			HolderFirstName: "Alice",
			HolderLastName:  "Johnson",
			HolderEmail:     "alice@example.com",
		}

		resp, err := uc.Execute(context.Background(), req)
		require.NoError(t, err)

		assert.Contains(t, resp.LedgerAccountCode, "2100-")
	})

	t.Run("rejects invalid account type", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, nil, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "INVALID",
			Currency:        "USD",
			HolderFirstName: "John",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		_, err := uc.Execute(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account type")
	})

	t.Run("rejects invalid holder data", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, nil, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		_, err := uc.Execute(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid holder data")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := &mockAccountRepository{saveErr: fmt.Errorf("database unavailable")}
		publisher := &mockEventPublisher{}
		ledger := &mockLedgerClient{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "John",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		_, err := uc.Execute(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save account")
	})

	t.Run("fails when ledger client fails", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		ledger := &mockLedgerClient{createErr: fmt.Errorf("ledger service unavailable")}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "John",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		_, err := uc.Execute(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create ledger account")
	})

	t.Run("succeeds even when event publishing fails", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{publishErr: fmt.Errorf("kafka unavailable")}
		ledger := &mockLedgerClient{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, ledger, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "John",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		resp, err := uc.Execute(context.Background(), req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.AccountID)
	})

	t.Run("works without ledger client (nil)", func(t *testing.T) {
		repo := &mockAccountRepository{}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewOpenAccountUseCase(repo, publisher, nil, logger)

		req := dto.OpenAccountRequest{
			TenantID:        uuid.New(),
			AccountType:     "CHECKING",
			Currency:        "USD",
			HolderFirstName: "John",
			HolderLastName:  "Doe",
			HolderEmail:     "john@example.com",
		}

		resp, err := uc.Execute(context.Background(), req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.AccountID)
	})
}
