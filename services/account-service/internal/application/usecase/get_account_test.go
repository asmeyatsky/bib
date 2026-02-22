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

func TestGetAccountUseCase_Execute(t *testing.T) {
	t.Run("successfully retrieves an account", func(t *testing.T) {
		accountID := uuid.New()
		tenantID := uuid.New()
		holder := model.ReconstructAccountHolder(uuid.New(), "Jane", "Smith", "jane@example.com", uuid.New())
		acctType, _ := valueobject.NewAccountType("CHECKING")
		now := time.Now()

		account := model.ReconstructCustomerAccount(
			accountID, tenantID, valueobject.NewAccountNumber(), acctType,
			model.AccountStatusActive, "USD", holder, "2000-100", 1, now, now,
		)

		repo := &mockAccountRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error) {
				return account, nil
			},
		}
		logger := testLogger()

		uc := usecase.NewGetAccountUseCase(repo, logger)

		req := dto.GetAccountRequest{AccountID: accountID}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, accountID, resp.AccountID)
		assert.Equal(t, tenantID, resp.TenantID)
		assert.Equal(t, "CHECKING", resp.AccountType)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.Equal(t, "USD", resp.Currency)
		assert.Equal(t, "2000-100", resp.LedgerAccountCode)
		assert.Equal(t, "Jane", resp.HolderFirstName)
		assert.Equal(t, "Smith", resp.HolderLastName)
		assert.Equal(t, "jane@example.com", resp.HolderEmail)
	})

	t.Run("fails when account not found", func(t *testing.T) {
		repo := &mockAccountRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error) {
				return model.CustomerAccount{}, fmt.Errorf("account not found")
			},
		}
		logger := testLogger()

		uc := usecase.NewGetAccountUseCase(repo, logger)

		req := dto.GetAccountRequest{AccountID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find account")
	})
}
