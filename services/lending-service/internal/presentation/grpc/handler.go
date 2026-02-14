package grpc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/application/usecase"
)

// ---------------------------------------------------------------------------
// LendingHandler exposes lending operations over gRPC.
// In a full implementation this would implement a protobuf-generated interface.
// Here we define a plain struct that can be easily wired into a gRPC server
// once the proto definitions are generated.
// ---------------------------------------------------------------------------

// LendingHandler is the gRPC handler for lending operations.
type LendingHandler struct {
	submitApp  *usecase.SubmitLoanApplicationUseCase
	disburse   *usecase.DisburseLoanUseCase
	payment    *usecase.MakePaymentUseCase
	getLoan    *usecase.GetLoanUseCase
	getApp     *usecase.GetApplicationUseCase
}

// NewLendingHandler creates a new handler with all use-case dependencies.
func NewLendingHandler(
	submitApp *usecase.SubmitLoanApplicationUseCase,
	disburse *usecase.DisburseLoanUseCase,
	payment *usecase.MakePaymentUseCase,
	getLoan *usecase.GetLoanUseCase,
	getApp *usecase.GetApplicationUseCase,
) *LendingHandler {
	return &LendingHandler{
		submitApp: submitApp,
		disburse:  disburse,
		payment:   payment,
		getLoan:   getLoan,
		getApp:    getApp,
	}
}

// SubmitApplication handles a new loan application submission.
func (h *LendingHandler) SubmitApplication(
	ctx context.Context,
	tenantID, applicantID string,
	requestedAmount string,
	currency string,
	termMonths int,
	purpose string,
) (dto.LoanApplicationResponse, error) {
	amount, err := decimal.NewFromString(requestedAmount)
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("invalid amount: %w", err)
	}

	return h.submitApp.Execute(ctx, dto.SubmitApplicationRequest{
		TenantID:        tenantID,
		ApplicantID:     applicantID,
		RequestedAmount: amount,
		Currency:        currency,
		TermMonths:      termMonths,
		Purpose:         purpose,
	})
}

// DisburseLoan handles loan disbursement for an approved application.
func (h *LendingHandler) DisburseLoan(
	ctx context.Context,
	tenantID, applicationID, borrowerAccountID string,
	interestRateBps int,
) (dto.LoanResponse, error) {
	return h.disburse.Execute(ctx, dto.DisburseLoanRequest{
		TenantID:          tenantID,
		ApplicationID:     applicationID,
		BorrowerAccountID: borrowerAccountID,
		InterestRateBps:   interestRateBps,
	})
}

// MakePayment handles a loan payment.
func (h *LendingHandler) MakePayment(
	ctx context.Context,
	tenantID, loanID, amount string,
) (dto.PaymentResponse, error) {
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return dto.PaymentResponse{}, fmt.Errorf("invalid amount: %w", err)
	}

	return h.payment.Execute(ctx, dto.MakePaymentRequest{
		TenantID: tenantID,
		LoanID:   loanID,
		Amount:   amt,
	})
}

// GetLoan retrieves a loan by ID.
func (h *LendingHandler) GetLoan(
	ctx context.Context,
	tenantID, loanID string,
) (dto.LoanResponse, error) {
	return h.getLoan.Execute(ctx, dto.GetLoanRequest{
		TenantID: tenantID,
		LoanID:   loanID,
	})
}

// GetApplication retrieves a loan application by ID.
func (h *LendingHandler) GetApplication(
	ctx context.Context,
	tenantID, applicationID string,
) (dto.LoanApplicationResponse, error) {
	return h.getApp.Execute(ctx, dto.GetApplicationRequest{
		TenantID:      tenantID,
		ApplicationID: applicationID,
	})
}
