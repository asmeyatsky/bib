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

func approvedApplication() model.LoanApplication {
	now := time.Now().UTC()
	return model.ReconstructLoanApplication(
		"app-001", "tenant-001", "applicant-001",
		decimal.NewFromInt(50000), "USD", 36, "home improvement",
		valueobject.LoanApplicationStatusApproved,
		"excellent credit tier", "750",
		2, now, now,
	)
}

func TestDisburseLoan_Execute(t *testing.T) {
	t.Run("successfully disburses a loan", func(t *testing.T) {
		app := approvedApplication()
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
				return app, nil
			},
		}
		loanRepo := &mockLoanRepository{}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)

		req := dto.DisburseLoanRequest{
			TenantID:          "tenant-001",
			ApplicationID:     "app-001",
			BorrowerAccountID: "account-001",
			InterestRateBps:   450,
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.True(t, decimal.NewFromInt(50000).Equal(resp.Principal))
		assert.Equal(t, "USD", resp.Currency)
		assert.Equal(t, 450, resp.InterestRateBps)
		assert.Equal(t, 36, resp.TermMonths)
		assert.NotEmpty(t, resp.Schedule)

		require.Len(t, appRepo.savedApps, 1)
		require.Len(t, loanRepo.savedLoans, 1)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("fails when application not found", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
				return model.LoanApplication{}, fmt.Errorf("application not found")
			},
		}
		loanRepo := &mockLoanRepository{}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)

		req := dto.DisburseLoanRequest{
			TenantID:          "tenant-001",
			ApplicationID:     "app-001",
			BorrowerAccountID: "account-001",
			InterestRateBps:   450,
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "find application")
	})

	t.Run("fails when application is not approved", func(t *testing.T) {
		now := time.Now().UTC()
		rejectedApp := model.ReconstructLoanApplication(
			"app-002", "tenant-001", "applicant-001",
			decimal.NewFromInt(50000), "USD", 36, "home improvement",
			valueobject.LoanApplicationStatusRejected,
			"credit score below minimum", "",
			2, now, now,
		)

		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
				return rejectedApp, nil
			},
		}
		loanRepo := &mockLoanRepository{}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)

		req := dto.DisburseLoanRequest{
			TenantID:          "tenant-001",
			ApplicationID:     "app-002",
			BorrowerAccountID: "account-001",
			InterestRateBps:   450,
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "mark disbursed")
	})

	t.Run("fails when loan save fails", func(t *testing.T) {
		app := approvedApplication()
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
				return app, nil
			},
		}
		loanRepo := &mockLoanRepository{
			saveFunc: func(ctx context.Context, loan model.Loan) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockLendingEventPublisher{}

		uc := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)

		req := dto.DisburseLoanRequest{
			TenantID:          "tenant-001",
			ApplicationID:     "app-001",
			BorrowerAccountID: "account-001",
			InterestRateBps:   450,
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "save loan")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		app := approvedApplication()
		appRepo := &mockLoanApplicationRepository{
			findByIDFunc: func(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
				return app, nil
			},
		}
		loanRepo := &mockLoanRepository{}
		publisher := &mockLendingEventPublisher{
			publishFunc: func(ctx context.Context, evts ...event.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}

		uc := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)

		req := dto.DisburseLoanRequest{
			TenantID:          "tenant-001",
			ApplicationID:     "app-001",
			BorrowerAccountID: "account-001",
			InterestRateBps:   450,
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish events")
	})
}
