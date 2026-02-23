package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/application/usecase"
	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/service"
)

// --- Mock implementations ---

type mockLoanApplicationRepository struct {
	saveFunc     func(ctx context.Context, app model.LoanApplication) error
	findByIDFunc func(ctx context.Context, tenantID, id string) (model.LoanApplication, error)
	savedApps    []model.LoanApplication
}

func (m *mockLoanApplicationRepository) Save(ctx context.Context, app model.LoanApplication) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, app)
	}
	m.savedApps = append(m.savedApps, app)
	return nil
}

func (m *mockLoanApplicationRepository) FindByID(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, tenantID, id)
	}
	return model.LoanApplication{}, fmt.Errorf("application not found")
}

func (m *mockLoanApplicationRepository) FindByApplicantID(_ context.Context, _, _ string) ([]model.LoanApplication, error) {
	return nil, nil
}

type mockLoanRepository struct {
	saveFunc     func(ctx context.Context, loan model.Loan) error
	findByIDFunc func(ctx context.Context, tenantID, id string) (model.Loan, error)
	savedLoans   []model.Loan
}

func (m *mockLoanRepository) Save(ctx context.Context, loan model.Loan) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, loan)
	}
	m.savedLoans = append(m.savedLoans, loan)
	return nil
}

func (m *mockLoanRepository) FindByID(ctx context.Context, tenantID, id string) (model.Loan, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, tenantID, id)
	}
	return model.Loan{}, fmt.Errorf("loan not found")
}

func (m *mockLoanRepository) FindByApplicationID(_ context.Context, _, _ string) (model.Loan, error) {
	return model.Loan{}, nil
}

func (m *mockLoanRepository) FindByBorrowerAccountID(_ context.Context, _, _ string) ([]model.Loan, error) {
	return nil, nil
}

type mockLendingEventPublisher struct {
	publishFunc     func(ctx context.Context, events ...event.DomainEvent) error
	publishedEvents []event.DomainEvent
}

func (m *mockLendingEventPublisher) Publish(ctx context.Context, evts ...event.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

type mockCreditBureauClient struct {
	getCreditScoreFunc func(ctx context.Context, applicantID string) (string, error)
}

func (m *mockCreditBureauClient) GetCreditScore(ctx context.Context, applicantID string) (string, error) {
	if m.getCreditScoreFunc != nil {
		return m.getCreditScoreFunc(ctx, applicantID)
	}
	return "750", nil
}

// --- Tests ---

func validSubmitRequest() dto.SubmitApplicationRequest {
	return dto.SubmitApplicationRequest{
		TenantID:        "tenant-001",
		ApplicantID:     "applicant-001",
		RequestedAmount: decimal.NewFromInt(50000),
		Currency:        "USD",
		TermMonths:      36,
		Purpose:         "home improvement",
	}
}

func TestSubmitLoanApplication_Execute(t *testing.T) {
	t.Run("successfully submits and approves a loan application", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{}
		publisher := &mockLendingEventPublisher{}
		creditClient := &mockCreditBureauClient{
			getCreditScoreFunc: func(_ context.Context, _ string) (string, error) {
				return "750", nil
			},
		}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "APPROVED", resp.Status)
		assert.NotEmpty(t, resp.DecisionReason)
		assert.Equal(t, "750", resp.CreditScore)

		require.Len(t, appRepo.savedApps, 1)
		assert.NotEmpty(t, publisher.publishedEvents)
	})

	t.Run("rejects application with low credit score", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{}
		publisher := &mockLendingEventPublisher{}
		creditClient := &mockCreditBureauClient{
			getCreditScoreFunc: func(_ context.Context, _ string) (string, error) {
				return "500", nil // below threshold
			},
		}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "REJECTED", resp.Status)
		assert.Contains(t, resp.DecisionReason, "credit score below minimum")
	})

	t.Run("fails with invalid request data", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{}
		publisher := &mockLendingEventPublisher{}
		creditClient := &mockCreditBureauClient{}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		req.TenantID = "" // invalid
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "create application")
	})

	t.Run("fails when credit bureau is unavailable", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{}
		publisher := &mockLendingEventPublisher{}
		creditClient := &mockCreditBureauClient{
			getCreditScoreFunc: func(_ context.Context, _ string) (string, error) {
				return "", fmt.Errorf("credit bureau unavailable")
			},
		}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch credit score")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{
			saveFunc: func(_ context.Context, _ model.LoanApplication) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockLendingEventPublisher{}
		creditClient := &mockCreditBureauClient{}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "save application")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		appRepo := &mockLoanApplicationRepository{}
		publisher := &mockLendingEventPublisher{
			publishFunc: func(_ context.Context, _ ...event.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}
		creditClient := &mockCreditBureauClient{}
		underwriter := service.NewUnderwritingEngine()

		uc := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)

		req := validSubmitRequest()
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish events")
	})
}
