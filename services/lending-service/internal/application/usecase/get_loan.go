package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/domain/port"
)

// GetLoanUseCase retrieves a loan by ID.
type GetLoanUseCase struct {
	loanRepo port.LoanRepository
}

// NewGetLoanUseCase wires dependencies.
func NewGetLoanUseCase(loanRepo port.LoanRepository) *GetLoanUseCase {
	return &GetLoanUseCase{loanRepo: loanRepo}
}

// Execute returns a loan response for the given ID.
func (uc *GetLoanUseCase) Execute(
	ctx context.Context,
	req dto.GetLoanRequest,
) (dto.LoanResponse, error) {
	loan, err := uc.loanRepo.FindByID(ctx, req.TenantID, req.LoanID)
	if err != nil {
		return dto.LoanResponse{}, fmt.Errorf("find loan: %w", err)
	}
	return toLoanResponse(loan), nil
}

// GetApplicationUseCase retrieves a loan application by ID.
type GetApplicationUseCase struct {
	appRepo port.LoanApplicationRepository
}

// NewGetApplicationUseCase wires dependencies.
func NewGetApplicationUseCase(appRepo port.LoanApplicationRepository) *GetApplicationUseCase {
	return &GetApplicationUseCase{appRepo: appRepo}
}

// Execute returns a loan application response for the given ID.
func (uc *GetApplicationUseCase) Execute(
	ctx context.Context,
	req dto.GetApplicationRequest,
) (dto.LoanApplicationResponse, error) {
	app, err := uc.appRepo.FindByID(ctx, req.TenantID, req.ApplicationID)
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("find application: %w", err)
	}
	return toApplicationResponse(app), nil
}
