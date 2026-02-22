package grpc

import (
	"context"
	"regexp"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/application/usecase"
)

var currencyCodeRE = regexp.MustCompile(`^[A-Z]{3}$`)

// requireRole checks that the caller has at least one of the given roles.
func requireRole(ctx context.Context, roles ...string) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	for _, role := range roles {
		if claims.HasRole(role) {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "insufficient permissions")
}

// tenantIDFromContext extracts the tenant ID as a string from JWT claims in the context.
func tenantIDFromContext(ctx context.Context) (string, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "authentication required")
	}
	return claims.TenantID.String(), nil
}

// ---------------------------------------------------------------------------
// LendingHandler exposes lending operations over gRPC.
// In a full implementation this would implement a protobuf-generated interface.
// Here we define a plain struct that can be easily wired into a gRPC server
// once the proto definitions are generated.
// ---------------------------------------------------------------------------

// LendingHandler is the gRPC handler for lending operations.
type LendingHandler struct {
	submitApp *usecase.SubmitLoanApplicationUseCase
	disburse  *usecase.DisburseLoanUseCase
	payment   *usecase.MakePaymentUseCase
	getLoan   *usecase.GetLoanUseCase
	getApp    *usecase.GetApplicationUseCase
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
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return dto.LoanApplicationResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.LoanApplicationResponse{}, err
	}

	if applicantID == "" {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "applicant_id is required")
	}
	amount, err := decimal.NewFromString(requestedAmount)
	if err != nil {
		return dto.LoanApplicationResponse{}, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amount.IsPositive() {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if currency == "" {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "currency is required")
	}
	if !currencyCodeRE.MatchString(currency) {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}
	if termMonths <= 0 {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "term_months must be positive")
	}

	resp, err := h.submitApp.Execute(ctx, dto.SubmitApplicationRequest{
		TenantID:        tid,
		ApplicantID:     applicantID,
		RequestedAmount: amount,
		Currency:        currency,
		TermMonths:      termMonths,
		Purpose:         purpose,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return dto.LoanApplicationResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// DisburseLoan handles loan disbursement for an approved application.
func (h *LendingHandler) DisburseLoan(
	ctx context.Context,
	tenantID, applicationID, borrowerAccountID string,
	interestRateBps int,
) (dto.LoanResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return dto.LoanResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.LoanResponse{}, err
	}

	if applicationID == "" {
		return dto.LoanResponse{}, status.Error(codes.InvalidArgument, "application_id is required")
	}
	if borrowerAccountID == "" {
		return dto.LoanResponse{}, status.Error(codes.InvalidArgument, "borrower_account_id is required")
	}
	if interestRateBps <= 0 {
		return dto.LoanResponse{}, status.Error(codes.InvalidArgument, "interest_rate_bps must be positive")
	}

	resp, err := h.disburse.Execute(ctx, dto.DisburseLoanRequest{
		TenantID:          tid,
		ApplicationID:     applicationID,
		BorrowerAccountID: borrowerAccountID,
		InterestRateBps:   interestRateBps,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return dto.LoanResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// MakePayment handles a loan payment.
func (h *LendingHandler) MakePayment(
	ctx context.Context,
	tenantID, loanID, amount string,
) (dto.PaymentResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return dto.PaymentResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.PaymentResponse{}, err
	}

	if loanID == "" {
		return dto.PaymentResponse{}, status.Error(codes.InvalidArgument, "loan_id is required")
	}
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return dto.PaymentResponse{}, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amt.IsPositive() {
		return dto.PaymentResponse{}, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	resp, err := h.payment.Execute(ctx, dto.MakePaymentRequest{
		TenantID: tid,
		LoanID:   loanID,
		Amount:   amt,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return dto.PaymentResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// GetLoan retrieves a loan by ID.
func (h *LendingHandler) GetLoan(
	ctx context.Context,
	tenantID, loanID string,
) (dto.LoanResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return dto.LoanResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.LoanResponse{}, err
	}

	if loanID == "" {
		return dto.LoanResponse{}, status.Error(codes.InvalidArgument, "loan_id is required")
	}

	resp, err := h.getLoan.Execute(ctx, dto.GetLoanRequest{
		TenantID: tid,
		LoanID:   loanID,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return dto.LoanResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// GetApplication retrieves a loan application by ID.
func (h *LendingHandler) GetApplication(
	ctx context.Context,
	tenantID, applicationID string,
) (dto.LoanApplicationResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return dto.LoanApplicationResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.LoanApplicationResponse{}, err
	}

	if applicationID == "" {
		return dto.LoanApplicationResponse{}, status.Error(codes.InvalidArgument, "application_id is required")
	}

	resp, err := h.getApp.Execute(ctx, dto.GetApplicationRequest{
		TenantID:      tid,
		ApplicationID: applicationID,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return dto.LoanApplicationResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}
