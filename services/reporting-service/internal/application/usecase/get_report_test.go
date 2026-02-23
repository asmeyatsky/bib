package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockReportSubmissionRepository struct {
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.ReportSubmission, error)
}

func (m *mockReportSubmissionRepository) Save(_ context.Context, _ model.ReportSubmission) error {
	return nil
}

func (m *mockReportSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (model.ReportSubmission, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.ReportSubmission{}, fmt.Errorf("report not found")
}

func (m *mockReportSubmissionRepository) FindByTenantAndPeriod(_ context.Context, _ uuid.UUID, _ string) ([]model.ReportSubmission, error) {
	return nil, nil
}

func (m *mockReportSubmissionRepository) FindByTenantAndType(_ context.Context, _ uuid.UUID, _ string) ([]model.ReportSubmission, error) {
	return nil, nil
}

// --- Tests ---

func TestGetReportUseCase_Execute(t *testing.T) {
	t.Run("successfully retrieves a report submission", func(t *testing.T) {
		tenantID := uuid.New()
		submissionID := uuid.New()
		now := time.Now().UTC()

		submission := model.Reconstruct(
			submissionID, tenantID,
			valueobject.ReportTypeCOREP, "2025-Q4",
			valueobject.SubmissionStatusDraft, "",
			nil, nil, []string{}, 1, now, now,
		)

		repo := &mockReportSubmissionRepository{
			findByIDFunc: func(_ context.Context, id uuid.UUID) (model.ReportSubmission, error) {
				assert.Equal(t, submissionID, id)
				return submission, nil
			},
		}

		uc := usecase.NewGetReportUseCase(repo)

		req := dto.GetReportRequest{ID: submissionID}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, submissionID, resp.ID)
		assert.Equal(t, tenantID, resp.TenantID)
		assert.Equal(t, "COREP", resp.ReportType)
		assert.Equal(t, "2025-Q4", resp.ReportingPeriod)
		assert.Equal(t, "DRAFT", resp.Status)
		assert.Equal(t, 1, resp.Version)
	})

	t.Run("retrieves a generated report with XBRL content", func(t *testing.T) {
		tenantID := uuid.New()
		submissionID := uuid.New()
		now := time.Now().UTC()
		genAt := now.Add(-time.Hour)

		submission := model.Reconstruct(
			submissionID, tenantID,
			valueobject.ReportTypeFINREP, "2025-Q3",
			valueobject.SubmissionStatusReady,
			"<?xml version=\"1.0\"?><xbrli:xbrl>...</xbrli:xbrl>",
			&genAt, nil, []string{}, 2, now, now,
		)

		repo := &mockReportSubmissionRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.ReportSubmission, error) {
				return submission, nil
			},
		}

		uc := usecase.NewGetReportUseCase(repo)

		req := dto.GetReportRequest{ID: submissionID}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "READY", resp.Status)
		assert.Equal(t, "FINREP", resp.ReportType)
		assert.NotEmpty(t, resp.XBRLContent)
		assert.NotNil(t, resp.GeneratedAt)
	})

	t.Run("fails when report not found", func(t *testing.T) {
		repo := &mockReportSubmissionRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.ReportSubmission, error) {
				return model.ReportSubmission{}, fmt.Errorf("report not found")
			},
		}

		uc := usecase.NewGetReportUseCase(repo)

		req := dto.GetReportRequest{ID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find report submission")
	})
}
