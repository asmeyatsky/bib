package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
)

// --- In-memory test doubles ---

type inMemoryRepo struct {
	submissions map[uuid.UUID]model.ReportSubmission
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		submissions: make(map[uuid.UUID]model.ReportSubmission),
	}
}

func (r *inMemoryRepo) Save(_ context.Context, submission model.ReportSubmission) error {
	r.submissions[submission.ID()] = submission
	return nil
}

func (r *inMemoryRepo) FindByID(_ context.Context, id uuid.UUID) (model.ReportSubmission, error) {
	s, ok := r.submissions[id]
	if !ok {
		return model.ReportSubmission{}, assert.AnError
	}
	return s, nil
}

func (r *inMemoryRepo) FindByTenantAndPeriod(_ context.Context, tenantID uuid.UUID, period string) ([]model.ReportSubmission, error) {
	var result []model.ReportSubmission
	for _, s := range r.submissions {
		if s.TenantID() == tenantID && s.ReportingPeriod() == period {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *inMemoryRepo) FindByTenantAndType(_ context.Context, tenantID uuid.UUID, reportType string) ([]model.ReportSubmission, error) {
	var result []model.ReportSubmission
	for _, s := range r.submissions {
		if s.TenantID() == tenantID && s.ReportType().String() == reportType {
			result = append(result, s)
		}
	}
	return result, nil
}

type mockEventPublisher struct {
	publishedEvents []event.DomainEvent
}

func (p *mockEventPublisher) Publish(_ context.Context, events ...event.DomainEvent) error {
	p.publishedEvents = append(p.publishedEvents, events...)
	return nil
}

type mockLedgerClient struct{}

func (c *mockLedgerClient) GetFinancialData(_ context.Context, tenantID uuid.UUID, period string) (service.ReportData, error) {
	return service.ReportData{
		TenantID:           tenantID,
		Period:             period,
		TotalAssets:        decimal.NewFromInt(1_000_000_000),
		TotalLiabilities:   decimal.NewFromInt(900_000_000),
		TotalEquity:        decimal.NewFromInt(100_000_000),
		NetIncome:          decimal.NewFromInt(15_000_000),
		RiskWeightedAssets: decimal.NewFromInt(600_000_000),
		CET1Ratio:          decimal.NewFromFloat(0.1600),
		LCRRatio:           decimal.NewFromFloat(1.3000),
	}, nil
}

// --- Tests ---

func TestGenerateReportUseCase_Execute(t *testing.T) {
	repo := newInMemoryRepo()
	publisher := &mockEventPublisher{}
	ledgerClient := &mockLedgerClient{}
	generator := service.NewXBRLGenerator()

	uc := usecase.NewGenerateReportUseCase(repo, publisher, ledgerClient, generator)
	ctx := context.Background()

	t.Run("generates COREP report successfully", func(t *testing.T) {
		tenantID := uuid.New()
		req := dto.GenerateReportRequest{
			TenantID:   tenantID,
			ReportType: "COREP",
			Period:     "2025-Q1",
		}

		resp, err := uc.Execute(ctx, req)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, resp.ID)
		assert.Equal(t, tenantID, resp.TenantID)
		assert.Equal(t, "COREP", resp.ReportType)
		assert.Equal(t, "2025-Q1", resp.ReportingPeriod)
		assert.Equal(t, "READY", resp.Status)
		assert.NotEmpty(t, resp.GeneratedAt)

		// Verify submission was persisted.
		saved, err := repo.FindByID(ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, "READY", saved.Status().String())
		assert.NotEmpty(t, saved.XBRLContent())
		assert.Contains(t, saved.XBRLContent(), "corep:")

		// Verify event was published.
		require.Len(t, publisher.publishedEvents, 1)
		genEvent, ok := publisher.publishedEvents[0].(event.ReportGenerated)
		require.True(t, ok)
		assert.Equal(t, resp.ID.String(), genEvent.AggregateID())
		assert.Equal(t, "COREP", genEvent.ReportType)
	})

	t.Run("generates FINREP report successfully", func(t *testing.T) {
		publisher.publishedEvents = nil // reset
		tenantID := uuid.New()
		req := dto.GenerateReportRequest{
			TenantID:   tenantID,
			ReportType: "FINREP",
			Period:     "2025-Q2",
		}

		resp, err := uc.Execute(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, "FINREP", resp.ReportType)
		assert.Equal(t, "2025-Q2", resp.ReportingPeriod)
		assert.Equal(t, "READY", resp.Status)

		saved, err := repo.FindByID(ctx, resp.ID)
		require.NoError(t, err)
		assert.Contains(t, saved.XBRLContent(), "finrep:")
	})

	t.Run("generates MREL report successfully", func(t *testing.T) {
		publisher.publishedEvents = nil
		tenantID := uuid.New()
		req := dto.GenerateReportRequest{
			TenantID:   tenantID,
			ReportType: "MREL",
			Period:     "2025-Q3",
		}

		resp, err := uc.Execute(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, "MREL", resp.ReportType)
		assert.Equal(t, "READY", resp.Status)

		saved, err := repo.FindByID(ctx, resp.ID)
		require.NoError(t, err)
		assert.Contains(t, saved.XBRLContent(), "mrel:")
	})

	t.Run("rejects invalid report type", func(t *testing.T) {
		req := dto.GenerateReportRequest{
			TenantID:   uuid.New(),
			ReportType: "INVALID",
			Period:     "2025-Q1",
		}

		_, err := uc.Execute(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid report type")
	})

	t.Run("rejects nil tenant ID", func(t *testing.T) {
		req := dto.GenerateReportRequest{
			TenantID:   uuid.Nil,
			ReportType: "COREP",
			Period:     "2025-Q1",
		}

		_, err := uc.Execute(ctx, req)
		assert.Error(t, err)
	})

	t.Run("rejects empty period", func(t *testing.T) {
		req := dto.GenerateReportRequest{
			TenantID:   uuid.New(),
			ReportType: "COREP",
			Period:     "",
		}

		_, err := uc.Execute(ctx, req)
		assert.Error(t, err)
	})
}
