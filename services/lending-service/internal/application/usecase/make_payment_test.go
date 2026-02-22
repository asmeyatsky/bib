package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/application/usecase"
	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

func activeLoan() model.Loan {
	now := time.Now().UTC()
	return model.ReconstructLoan(
		"loan-001", "tenant-001", "app-001", "account-001",
		decimal.NewFromInt(10000), "USD", 450, 12,
		valueobject.LoanStatusActive,
		[]model.AmortizationEntry{},
		decimal.NewFromInt(10000),
		now.AddDate(0, 1, 0),
		1, now, now,
	)
}

func TestMakePayment_Execute(t *testing.T) {
	t.Run("successfully makes a payment", func(t *testing.T) {
		loan := activeLoan()
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return loan, nil
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{
			TenantID: "tenant-001",
			LoanID:   "loan-001",
			Amount:   decimal.NewFromInt(1000),
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "loan-001", resp.LoanID)
		assert.True(t, decimal.NewFromInt(1000).Equal(resp.AmountPaid))
		assert.True(t, decimal.NewFromInt(9000).Equal(resp.OutstandingBalance))
		assert.Equal(t, "ACTIVE", resp.LoanStatus)

		require.Len(t, loanRepo.savedLoans, 1)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("pays off loan completely", func(t *testing.T) {
		loan := activeLoan()
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return loan, nil
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{
			TenantID: "tenant-001",
			LoanID:   "loan-001",
			Amount:   decimal.NewFromInt(10000), // full payoff
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "PAID_OFF", resp.LoanStatus)
		assert.True(t, decimal.Zero.Equal(resp.OutstandingBalance))
	})

	t.Run("fails when loan not found", func(t *testing.T) {
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return model.Loan{}, fmt.Errorf("loan not found")
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{TenantID: "tenant-001", LoanID: "loan-001", Amount: decimal.NewFromInt(100)}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "find loan")
	})

	t.Run("fails when payment exceeds balance", func(t *testing.T) {
		loan := activeLoan()
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return loan, nil
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{
			TenantID: "tenant-001",
			LoanID:   "loan-001",
			Amount:   decimal.NewFromInt(50000), // exceeds balance
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "make payment")
	})

	t.Run("fails when loan save fails", func(t *testing.T) {
		loan := activeLoan()
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return loan, nil
			},
			saveFunc: func(ctx context.Context, l model.Loan) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{TenantID: "tenant-001", LoanID: "loan-001", Amount: decimal.NewFromInt(100)}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "save loan")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		loan := activeLoan()
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.Loan, error) {
				return loan, nil
			},
		}
		publisher := &mockLendingEventPublisher{
			publishFunc: func(ctx context.Context, evts ...event.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}

		uc := usecase.NewMakePaymentUseCase(loanRepo, publisher)

		req := dto.MakePaymentRequest{TenantID: "tenant-001", LoanID: "loan-001", Amount: decimal.NewFromInt(100)}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish events")
	})
}
