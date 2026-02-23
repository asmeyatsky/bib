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

func TestCloseAccountUseCase_Execute(t *testing.T) {
	t.Run("successfully closes an active account", func(t *testing.T) {
		account := activeAccount()
		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return account, nil
			},
		}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{
			AccountID: account.ID(),
			Reason:    "customer request",
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, account.ID(), resp.AccountID)
		assert.Equal(t, "CLOSED", resp.Status)

		// Verify repository was called with the closed account.
		require.NotNil(t, repo.savedAccount)
		assert.Equal(t, model.AccountStatusClosed, repo.savedAccount.Status())

		// Verify events were published.
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("successfully closes a frozen account", func(t *testing.T) {
		holder := model.ReconstructAccountHolder(uuid.New(), "Jane", "Smith", "jane@example.com", uuid.New())
		acctType, _ := valueobject.NewAccountType("CHECKING")
		now := time.Now()
		frozenAccount := model.ReconstructCustomerAccount(
			uuid.New(), uuid.New(), valueobject.NewAccountNumber(), acctType,
			model.AccountStatusFrozen, "USD", holder, "2000-100", 2, now, now,
		)

		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return frozenAccount, nil
			},
		}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{AccountID: frozenAccount.ID(), Reason: "compliance"}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "CLOSED", resp.Status)
	})

	t.Run("fails when account not found", func(t *testing.T) {
		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return model.CustomerAccount{}, fmt.Errorf("account not found")
			},
		}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{AccountID: uuid.New(), Reason: "test"}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find account")
	})

	t.Run("fails when account is pending", func(t *testing.T) {
		holder := model.ReconstructAccountHolder(uuid.New(), "Jane", "Smith", "jane@example.com", uuid.New())
		acctType, _ := valueobject.NewAccountType("CHECKING")
		now := time.Now()
		pendingAccount := model.ReconstructCustomerAccount(
			uuid.New(), uuid.New(), valueobject.NewAccountNumber(), acctType,
			model.AccountStatusPending, "USD", holder, "2000-100", 1, now, now,
		)

		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return pendingAccount, nil
			},
		}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{AccountID: pendingAccount.ID(), Reason: "test"}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close account")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		account := activeAccount()
		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return account, nil
			},
			saveErr: fmt.Errorf("database unavailable"),
		}
		publisher := &mockEventPublisher{}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{AccountID: account.ID(), Reason: "customer request"}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save closed account")
	})

	t.Run("succeeds even when event publishing fails", func(t *testing.T) {
		account := activeAccount()
		repo := &mockAccountRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.CustomerAccount, error) {
				return account, nil
			},
		}
		publisher := &mockEventPublisher{publishErr: fmt.Errorf("kafka unavailable")}
		logger := testLogger()

		uc := usecase.NewCloseAccountUseCase(repo, publisher, logger)

		req := dto.CloseAccountRequest{AccountID: account.ID(), Reason: "customer request"}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "CLOSED", resp.Status)
	})
}
