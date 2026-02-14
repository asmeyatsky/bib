package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
)

// ListPayments handles listing payment orders with pagination.
type ListPayments struct {
	paymentRepo port.PaymentOrderRepository
}

func NewListPayments(paymentRepo port.PaymentOrderRepository) *ListPayments {
	return &ListPayments{paymentRepo: paymentRepo}
}

func (uc *ListPayments) Execute(ctx context.Context, req dto.ListPaymentsRequest) (dto.ListPaymentsResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var (
		orders []model.PaymentOrder
		total  int
		err    error
	)

	if req.AccountID != uuid.Nil {
		orders, total, err = uc.paymentRepo.ListByAccount(ctx, req.AccountID, pageSize, req.Offset)
	} else {
		orders, total, err = uc.paymentRepo.ListByTenant(ctx, req.TenantID, pageSize, req.Offset)
	}
	if err != nil {
		return dto.ListPaymentsResponse{}, fmt.Errorf("failed to list payment orders: %w", err)
	}

	var responses []dto.PaymentOrderResponse
	for _, order := range orders {
		responses = append(responses, toPaymentOrderResponse(order))
	}

	return dto.ListPaymentsResponse{
		Payments:   responses,
		TotalCount: total,
	}, nil
}
