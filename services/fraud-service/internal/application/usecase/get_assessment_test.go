package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/valueobject"
)

func TestGetAssessment_Execute(t *testing.T) {
	t.Run("successfully retrieves an assessment", func(t *testing.T) {
		tenantID := uuid.New()
		assessmentID := uuid.New()
		now := time.Now().UTC()

		assessment := model.Reconstruct(
			assessmentID, tenantID, uuid.New(), uuid.New(),
			decimal.NewFromInt(1000), "USD", "transfer",
			valueobject.RiskLevelLow, 10, valueobject.DecisionApprove,
			[]string{}, now, 1, now, now,
		)

		repo := &mockAssessmentRepository{
			findByIDFunc: func(_ context.Context, tid, id uuid.UUID) (*model.TransactionAssessment, error) {
				assert.Equal(t, tenantID, tid)
				assert.Equal(t, assessmentID, id)
				return assessment, nil
			},
		}

		uc := usecase.NewGetAssessment(repo)

		req := dto.GetAssessmentRequest{TenantID: tenantID, AssessmentID: assessmentID}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, assessmentID, resp.ID)
		assert.Equal(t, tenantID, resp.TenantID)
		assert.Equal(t, "LOW", resp.RiskLevel)
		assert.Equal(t, "APPROVE", resp.Decision)
	})

	t.Run("fails when assessment not found", func(t *testing.T) {
		repo := &mockAssessmentRepository{
			findByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*model.TransactionAssessment, error) {
				return nil, fmt.Errorf("not found")
			},
		}

		uc := usecase.NewGetAssessment(repo)

		req := dto.GetAssessmentRequest{TenantID: uuid.New(), AssessmentID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find assessment")
	})

	t.Run("fails when assessment is nil", func(t *testing.T) {
		repo := &mockAssessmentRepository{
			findByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*model.TransactionAssessment, error) {
				return nil, nil
			},
		}

		uc := usecase.NewGetAssessment(repo)

		req := dto.GetAssessmentRequest{TenantID: uuid.New(), AssessmentID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "assessment not found")
	})
}
