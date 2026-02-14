package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/port"
)

// DisburseLoanUseCase creates a Loan from an approved application, generates
// the amortization schedule, and publishes the disbursement event to the ledger.
type DisburseLoanUseCase struct {
	appRepo   port.LoanApplicationRepository
	loanRepo  port.LoanRepository
	publisher port.EventPublisher
}

// NewDisburseLoanUseCase wires dependencies.
func NewDisburseLoanUseCase(
	appRepo port.LoanApplicationRepository,
	loanRepo port.LoanRepository,
	publisher port.EventPublisher,
) *DisburseLoanUseCase {
	return &DisburseLoanUseCase{
		appRepo:   appRepo,
		loanRepo:  loanRepo,
		publisher: publisher,
	}
}

// Execute disburses a loan for an approved application.
func (uc *DisburseLoanUseCase) Execute(
	ctx context.Context,
	req dto.DisburseLoanRequest,
) (dto.LoanResponse, error) {
	now := time.Now().UTC()

	// 1. Retrieve the approved application.
	app, err := uc.appRepo.FindByID(ctx, req.TenantID, req.ApplicationID)
	if err != nil {
		return dto.LoanResponse{}, fmt.Errorf("find application: %w", err)
	}

	// 2. Mark application as disbursed.
	app, err = app.MarkDisbursed(now)
	if err != nil {
		return dto.LoanResponse{}, fmt.Errorf("mark disbursed: %w", err)
	}
	if err := uc.appRepo.Save(ctx, app); err != nil {
		return dto.LoanResponse{}, fmt.Errorf("save application: %w", err)
	}

	// 3. Create the Loan aggregate (generates schedule internally).
	loan, err := model.NewLoan(
		req.TenantID, req.ApplicationID, req.BorrowerAccountID,
		app.RequestedAmount(), app.Currency(),
		req.InterestRateBps, app.TermMonths(), now,
	)
	if err != nil {
		return dto.LoanResponse{}, fmt.Errorf("create loan: %w", err)
	}

	// 4. Persist the loan.
	if err := uc.loanRepo.Save(ctx, loan); err != nil {
		return dto.LoanResponse{}, fmt.Errorf("save loan: %w", err)
	}

	// 5. Publish domain events (LoanDisbursed -> ledger).
	if err := uc.publisher.Publish(ctx, loan.DomainEvents()...); err != nil {
		return dto.LoanResponse{}, fmt.Errorf("publish events: %w", err)
	}

	return toLoanResponse(loan), nil
}

func toLoanResponse(loan model.Loan) dto.LoanResponse {
	sched := loan.Schedule()
	entries := make([]dto.AmortizationEntryResponse, len(sched))
	for i, e := range sched {
		entries[i] = dto.AmortizationEntryResponse{
			Period:           e.Period,
			DueDate:          e.DueDate,
			Principal:        e.Principal,
			Interest:         e.Interest,
			Total:            e.Total,
			RemainingBalance: e.RemainingBalance,
		}
	}

	return dto.LoanResponse{
		ID:                 loan.ID(),
		TenantID:           loan.TenantID(),
		ApplicationID:      loan.ApplicationID(),
		BorrowerAccountID:  loan.BorrowerAccountID(),
		Principal:          loan.Principal(),
		Currency:           loan.Currency(),
		InterestRateBps:    loan.InterestRateBps(),
		TermMonths:         loan.TermMonths(),
		Status:             loan.Status().String(),
		OutstandingBalance: loan.OutstandingBalance(),
		NextPaymentDue:     loan.NextPaymentDue(),
		Schedule:           entries,
		CreatedAt:          loan.CreatedAt(),
		UpdatedAt:          loan.UpdatedAt(),
	}
}
