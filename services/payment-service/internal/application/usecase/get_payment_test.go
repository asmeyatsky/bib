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

	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

func samplePaymentOrder() model.PaymentOrder {
	now := time.Now().UTC()
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	return model.Reconstruct(
		uuid.New(), uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(1000), "USD",
		valueobject.RailACH, valueobject.PaymentStatusInitiated,
		routingInfo, "PAY-001", "ACH payment", "",
		now, nil, 1, now, now,
	)
}

func TestGetPayment_Execute(t *testing.T) {
	t.Run("successfully retrieves a payment order", func(t *testing.T) {
		order := samplePaymentOrder()

		repo := &mockPaymentOrderRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.PaymentOrder, error) {
				return order, nil
			},
		}

		uc := usecase.NewGetPayment(repo)

		req := dto.GetPaymentRequest{PaymentID: order.ID()}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, order.ID(), resp.ID)
		assert.Equal(t, order.TenantID(), resp.TenantID)
		assert.Equal(t, "ACH", resp.Rail)
		assert.Equal(t, "INITIATED", resp.Status)
		assert.True(t, decimal.NewFromInt(1000).Equal(resp.Amount))
		assert.Equal(t, "USD", resp.Currency)
		assert.Equal(t, "PAY-001", resp.Reference)
		assert.Equal(t, "ACH payment", resp.Description)
	})

	t.Run("fails when payment order not found", func(t *testing.T) {
		repo := &mockPaymentOrderRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (model.PaymentOrder, error) {
				return model.PaymentOrder{}, fmt.Errorf("payment order not found")
			},
		}

		uc := usecase.NewGetPayment(repo)

		req := dto.GetPaymentRequest{PaymentID: uuid.New()}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find payment order")
	})
}
