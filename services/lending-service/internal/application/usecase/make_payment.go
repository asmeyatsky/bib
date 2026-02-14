package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/domain/port"
)

// MakePaymentUseCase applies a payment to an outstanding loan.
type MakePaymentUseCase struct {
	loanRepo  port.LoanRepository
	publisher port.EventPublisher
}

// NewMakePaymentUseCase wires dependencies.
func NewMakePaymentUseCase(
	loanRepo port.LoanRepository,
	publisher port.EventPublisher,
) *MakePaymentUseCase {
	return &MakePaymentUseCase{
		loanRepo:  loanRepo,
		publisher: publisher,
	}
}

// Execute processes a payment against a loan.
func (uc *MakePaymentUseCase) Execute(
	ctx context.Context,
	req dto.MakePaymentRequest,
) (dto.PaymentResponse, error) {
	now := time.Now().UTC()

	// 1. Retrieve the loan.
	loan, err := uc.loanRepo.FindByID(ctx, req.TenantID, req.LoanID)
	if err != nil {
		return dto.PaymentResponse{}, fmt.Errorf("find loan: %w", err)
	}

	// 2. Apply payment.
	loan, err = loan.MakePayment(req.Amount, now)
	if err != nil {
		return dto.PaymentResponse{}, fmt.Errorf("make payment: %w", err)
	}

	// 3. Persist updated loan.
	if err := uc.loanRepo.Save(ctx, loan); err != nil {
		return dto.PaymentResponse{}, fmt.Errorf("save loan: %w", err)
	}

	// 4. Publish events.
	if err := uc.publisher.Publish(ctx, loan.DomainEvents()...); err != nil {
		return dto.PaymentResponse{}, fmt.Errorf("publish events: %w", err)
	}

	return dto.PaymentResponse{
		LoanID:             loan.ID(),
		AmountPaid:         req.Amount,
		OutstandingBalance: loan.OutstandingBalance(),
		LoanStatus:         loan.Status().String(),
	}, nil
}
