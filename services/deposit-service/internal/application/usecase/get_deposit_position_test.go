package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/deposit-service/internal/application/dto"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
)

func TestGetDepositPosition_Execute(t *testing.T) {
	t.Run("successfully retrieves a deposit position", func(t *testing.T) {
		position, _ := model.NewDepositPosition(
			uuid.New(), uuid.New(), uuid.New(),
			decimal.NewFromInt(5000), "USD", nil,
		)

		positionRepo := &mockDepositPositionRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositPosition, error) {
				return position, nil
			},
		}

		uc := usecase.NewGetDepositPosition(positionRepo)

		req := dto.GetPositionRequest{PositionID: position.ID()}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, position.ID(), resp.ID)
		assert.Equal(t, "ACTIVE", resp.Status)
		assert.True(t, decimal.NewFromInt(5000).Equal(resp.Principal))
		assert.Equal(t, "USD", resp.Currency)
	})

	t.Run("fails when position not found", func(t *testing.T) {
		positionRepo := &mockDepositPositionRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.DepositPosition, error) {
				return model.DepositPosition{}, fmt.Errorf("position not found")
			},
		}

		uc := usecase.NewGetDepositPosition(positionRepo)

		req := dto.GetPositionRequest{PositionID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find position")
	})
}
