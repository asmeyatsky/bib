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
// Request / Response types (stand-in for proto-generated messages)
// ---------------------------------------------------------------------------

// SubmitApplicationRequest represents the proto SubmitApplicationRequest message.
type SubmitApplicationRequest struct {
	TenantID        string `json:"tenant_id"`
	ApplicantID     string `json:"applicant_id"`
	RequestedAmount string `json:"requested_amount"`
	Currency        string `json:"currency"`
	Purpose         string `json:"purpose"`
	TermMonths      int    `json:"term_months"`
}

// SubmitApplicationResponse represents the proto SubmitApplicationResponse message.
type SubmitApplicationResponse = dto.LoanApplicationResponse

// DisburseLoanRequest represents the proto DisburseLoanRequest message.
type DisburseLoanRequest struct {
	TenantID          string `json:"tenant_id"`
	ApplicationID     string `json:"application_id"`
	BorrowerAccountID string `json:"borrower_account_id"`
	InterestRateBps   int    `json:"interest_rate_bps"`
}

// DisburseLoanResponse represents the proto DisburseLoanResponse message.
type DisburseLoanResponse = dto.LoanResponse

// MakePaymentRequest represents the proto MakePaymentRequest message.
type MakePaymentRequest struct {
	TenantID string `json:"tenant_id"`
	LoanID   string `json:"loan_id"`
	Amount   string `json:"amount"`
}

// MakePaymentResponse represents the proto MakePaymentResponse message.
type MakePaymentResponse = dto.PaymentResponse

// GetLoanRequest represents the proto GetLoanRequest message.
type GetLoanRequest struct {
	TenantID string `json:"tenant_id"`
	LoanID   string `json:"loan_id"`
}

// GetLoanResponse represents the proto GetLoanResponse message.
type GetLoanResponse = dto.LoanResponse

// GetApplicationRequest represents the proto GetApplicationRequest message.
type GetApplicationRequest struct {
	TenantID      string `json:"tenant_id"`
	ApplicationID string `json:"application_id"`
}

// GetApplicationResponse represents the proto GetApplicationResponse message.
type GetApplicationResponse = dto.LoanApplicationResponse

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
func (h *LendingHandler) SubmitApplication(ctx context.Context, req *SubmitApplicationRequest) (*SubmitApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.ApplicantID == "" {
		return nil, status.Error(codes.InvalidArgument, "applicant_id is required")
	}
	amount, err := decimal.NewFromString(req.RequestedAmount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amount.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if !currencyCodeRE.MatchString(req.Currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}
	if req.TermMonths <= 0 {
		return nil, status.Error(codes.InvalidArgument, "term_months must be positive")
	}

	resp, err := h.submitApp.Execute(ctx, dto.SubmitApplicationRequest{
		TenantID:        tid,
		ApplicantID:     req.ApplicantID,
		RequestedAmount: amount,
		Currency:        req.Currency,
		TermMonths:      req.TermMonths,
		Purpose:         req.Purpose,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &resp, nil
}

// DisburseLoan handles loan disbursement for an approved application.
func (h *LendingHandler) DisburseLoan(ctx context.Context, req *DisburseLoanRequest) (*DisburseLoanResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.ApplicationID == "" {
		return nil, status.Error(codes.InvalidArgument, "application_id is required")
	}
	if req.BorrowerAccountID == "" {
		return nil, status.Error(codes.InvalidArgument, "borrower_account_id is required")
	}
	if req.InterestRateBps <= 0 {
		return nil, status.Error(codes.InvalidArgument, "interest_rate_bps must be positive")
	}

	resp, err := h.disburse.Execute(ctx, dto.DisburseLoanRequest{
		TenantID:          tid,
		ApplicationID:     req.ApplicationID,
		BorrowerAccountID: req.BorrowerAccountID,
		InterestRateBps:   req.InterestRateBps,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &resp, nil
}

// MakePayment handles a loan payment.
func (h *LendingHandler) MakePayment(ctx context.Context, req *MakePaymentRequest) (*MakePaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.LoanID == "" {
		return nil, status.Error(codes.InvalidArgument, "loan_id is required")
	}
	amt, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amt.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	resp, err := h.payment.Execute(ctx, dto.MakePaymentRequest{
		TenantID: tid,
		LoanID:   req.LoanID,
		Amount:   amt,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &resp, nil
}

// GetLoan retrieves a loan by ID.
func (h *LendingHandler) GetLoan(ctx context.Context, req *GetLoanRequest) (*GetLoanResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.LoanID == "" {
		return nil, status.Error(codes.InvalidArgument, "loan_id is required")
	}

	resp, err := h.getLoan.Execute(ctx, dto.GetLoanRequest{
		TenantID: tid,
		LoanID:   req.LoanID,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &resp, nil
}

// GetApplication retrieves a loan application by ID.
func (h *LendingHandler) GetApplication(ctx context.Context, req *GetApplicationRequest) (*GetApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.ApplicationID == "" {
		return nil, status.Error(codes.InvalidArgument, "application_id is required")
	}

	resp, err := h.getApp.Execute(ctx, dto.GetApplicationRequest{
		TenantID:      tid,
		ApplicationID: req.ApplicationID,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &resp, nil
}
