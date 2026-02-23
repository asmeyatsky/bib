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
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

func TestGetLoanUseCase_Execute(t *testing.T) {
	t.Run("successfully retrieves a loan", func(t *testing.T) {
		now := time.Now().UTC()
		loan := model.ReconstructLoan(
			"loan-001", "tenant-001", "app-001", "account-001",
			decimal.NewFromInt(50000), "USD", 450, 36,
			valueobject.LoanStatusActive,
			[]model.AmortizationEntry{},
			decimal.NewFromInt(50000),
			now.AddDate(0, 1, 0),
			1, now, now,
		)

		loanRepo := &mockLoanRepository{
			findByIDFunc: func(_ context.Context, tenantID, id string) (model.Loan, error) {
				assert.Equal(t, "tenant-001", tenantID)
				assert.Equal(t, "loan-001", id)
				return loan, nil
			},
		}

		uc := usecase.NewGetLoanUseCase(loanRepo)

		req := dto.GetLoanRequest{TenantID: "tenant-001", LoanID: "loan-001"}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "loan-001", resp.ID)
		assert.Equal(t, "tenant-001", resp.TenantID)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.True(t, decimal.NewFromInt(50000).Equal(resp.Principal))
	})

	t.Run("fails when loan not found", func(t *testing.T) {
		loanRepo := &mockLoanRepository{
			findByIDFunc: func(_ context.Context, _, _ string) (model.Loan, error) {
				return model.Loan{}, fmt.Errorf("loan not found")
			},
		}

		uc := usecase.NewGetLoanUseCase(loanRepo)

		req := dto.GetLoanRequest{TenantID: "tenant-001", LoanID: "loan-999"}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "find loan")
	})
}

func TestGetApplicationUseCase_Execute(t *testing.T) {
	t.Run("successfully retrieves an application", func(t *testing.T) {
		app := approvedApplication()
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(_ context.Context, _, _ string) (model.LoanApplication, error) {
				return app, nil
			},
		}

		uc := usecase.NewGetApplicationUseCase(appRepo)

		req := dto.GetApplicationRequest{TenantID: "tenant-001", ApplicationID: "app-001"}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "app-001", resp.ID)
		assert.Equal(t, "APPROVED", resp.Status)
	})

	t.Run("fails when application not found", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(_ context.Context, _, _ string) (model.LoanApplication, error) {
				return model.LoanApplication{}, fmt.Errorf("not found")
			},
		}

		uc := usecase.NewGetApplicationUseCase(appRepo)

		req := dto.GetApplicationRequest{TenantID: "tenant-001", ApplicationID: "app-999"}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "find application")
	})
}
