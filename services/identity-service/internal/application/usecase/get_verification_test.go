package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/application/usecase"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
)

func TestGetVerification_Execute(t *testing.T) {
	t.Run("successfully retrieves a verification", func(t *testing.T) {
		v, _ := model.NewIdentityVerification(
			uuid.New(), "Jane", "Smith", "jane@example.com", "1990-01-01", "US",
		)

		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
		}

		uc := usecase.NewGetVerification(repo)

		req := dto.GetVerificationRequest{ID: v.ID()}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, v.ID(), resp.ID)
		assert.Equal(t, v.TenantID(), resp.TenantID)
		assert.Equal(t, "Jane", resp.ApplicantFirstName)
		assert.Equal(t, "Smith", resp.ApplicantLastName)
		assert.Equal(t, "jane@example.com", resp.ApplicantEmail)
		assert.Equal(t, "PENDING", resp.Status)
		assert.NotEmpty(t, resp.Checks)
	})

	t.Run("fails when verification not found", func(t *testing.T) {
		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return model.IdentityVerification{}, fmt.Errorf("not found")
			},
		}

		uc := usecase.NewGetVerification(repo)

		req := dto.GetVerificationRequest{ID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find verification")
	})
}
