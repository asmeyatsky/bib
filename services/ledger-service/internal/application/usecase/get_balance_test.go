package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func TestGetBalance_Execute(t *testing.T) {
	t.Run("successfully retrieves balance", func(t *testing.T) {
		balanceRepo := &mockBalanceRepository{
			getBalanceFunc: func(_ context.Context, account valueobject.AccountCode, currency string, _ time.Time) (decimal.Decimal, error) {
				assert.Equal(t, "1000", account.Code())
				assert.Equal(t, "USD", currency)
				return decimal.NewFromInt(5000), nil
			},
		}

		uc := usecase.NewGetBalance(balanceRepo)

		req := dto.GetBalanceRequest{
			AccountCode: "1000",
			Currency:    "USD",
			AsOf:        time.Now().UTC(),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "1000", resp.AccountCode)
		assert.True(t, decimal.NewFromInt(5000).Equal(resp.Amount))
		assert.Equal(t, "USD", resp.Currency)
	})

	t.Run("uses current time when AsOf is zero", func(t *testing.T) {
		balanceRepo := &mockBalanceRepository{
			getBalanceFunc: func(_ context.Context, _ valueobject.AccountCode, _ string, asOf time.Time) (decimal.Decimal, error) {
				assert.False(t, asOf.IsZero())
				return decimal.NewFromInt(100), nil
			},
		}

		uc := usecase.NewGetBalance(balanceRepo)

		req := dto.GetBalanceRequest{
			AccountCode: "1000",
			Currency:    "USD",
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(100).Equal(resp.Amount))
	})

	t.Run("fails with invalid account code", func(t *testing.T) {
		balanceRepo := &mockBalanceRepository{}

		uc := usecase.NewGetBalance(balanceRepo)

		req := dto.GetBalanceRequest{
			AccountCode: "INVALID",
			Currency:    "USD",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account code")
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		balanceRepo := &mockBalanceRepository{
			getBalanceFunc: func(_ context.Context, _ valueobject.AccountCode, _ string, _ time.Time) (decimal.Decimal, error) {
				return decimal.Zero, fmt.Errorf("database unavailable")
			},
		}

		uc := usecase.NewGetBalance(balanceRepo)

		req := dto.GetBalanceRequest{
			AccountCode: "1000",
			Currency:    "USD",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get balance")
	})
}
