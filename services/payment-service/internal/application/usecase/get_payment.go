package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
)

// GetPayment handles retrieval of a single payment order by ID.
type GetPayment struct {
	paymentRepo port.PaymentOrderRepository
}

func NewGetPayment(paymentRepo port.PaymentOrderRepository) *GetPayment {
	return &GetPayment{paymentRepo: paymentRepo}
}

func (uc *GetPayment) Execute(ctx context.Context, req dto.GetPaymentRequest) (dto.PaymentOrderResponse, error) {
	order, err := uc.paymentRepo.FindByID(ctx, req.PaymentID)
	if err != nil {
		return dto.PaymentOrderResponse{}, fmt.Errorf("failed to find payment order: %w", err)
	}
	return toPaymentOrderResponse(order), nil
}

func toPaymentOrderResponse(order model.PaymentOrder) dto.PaymentOrderResponse {
	return dto.PaymentOrderResponse{
		ID:                    order.ID(),
		TenantID:              order.TenantID(),
		SourceAccountID:       order.SourceAccountID(),
		DestinationAccountID:  order.DestinationAccountID(),
		Amount:                order.Amount(),
		Currency:              order.Currency(),
		Rail:                  order.Rail().String(),
		Status:                order.Status().String(),
		RoutingNumber:         order.RoutingInfo().RoutingNumber(),
		ExternalAccountNumber: order.RoutingInfo().ExternalAccountNumber(),
		Reference:             order.Reference(),
		Description:           order.Description(),
		FailureReason:         order.FailureReason(),
		InitiatedAt:           order.InitiatedAt(),
		SettledAt:             order.SettledAt(),
		Version:               order.Version(),
		CreatedAt:             order.CreatedAt(),
		UpdatedAt:             order.UpdatedAt(),
	}
}
