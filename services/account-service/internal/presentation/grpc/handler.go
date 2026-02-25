package grpc

import (
	"context"
	"log/slog"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
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

// tenantIDFromContext extracts the tenant ID from JWT claims in the context.
func tenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return claims.TenantID, nil
}

// Compile-time assertion that AccountHandler implements AccountServiceServer.
var _ AccountServiceServer = (*AccountHandler)(nil)

// AccountHandler implements the gRPC AccountServiceServer interface.
type AccountHandler struct {
	UnimplementedAccountServiceServer
	openAccount   *usecase.OpenAccountUseCase
	getAccount    *usecase.GetAccountUseCase
	freezeAccount *usecase.FreezeAccountUseCase
	closeAccount  *usecase.CloseAccountUseCase
	listAccounts  *usecase.ListAccountsUseCase

	logger               *slog.Logger}

// NewAccountHandler creates a new gRPC account handler.
func NewAccountHandler(
	openAccount *usecase.OpenAccountUseCase,
	getAccount *usecase.GetAccountUseCase,
	freezeAccount *usecase.FreezeAccountUseCase,
	closeAccount *usecase.CloseAccountUseCase,
	listAccounts *usecase.ListAccountsUseCase,
	logger *slog.Logger,
) *AccountHandler {
	return &AccountHandler{
		openAccount:   openAccount,
		getAccount:    getAccount,
		freezeAccount: freezeAccount,
		closeAccount:  closeAccount,
		listAccounts:  listAccounts,
	
		logger:               logger,}
}

// OpenAccountRequest represents the proto OpenAccountRequest message.
type OpenAccountRequest struct {
	TenantID               string `json:"tenant_id"`
	AccountType            string `json:"account_type"`
	Currency               string `json:"currency"`
	HolderFirstName        string `json:"holder_first_name"`
	HolderLastName         string `json:"holder_last_name"`
	HolderEmail            string `json:"holder_email"`
	IdentityVerificationID string `json:"identity_verification_id"`
}

// OpenAccountResponse represents the proto OpenAccountResponse message.
type OpenAccountResponse struct {
	AccountID         string `json:"account_id"`
	AccountNumber     string `json:"account_number"`
	Status            string `json:"status"`
	LedgerAccountCode string `json:"ledger_account_code"`
}

// GetAccountRequest represents the proto GetAccountRequest message.
type GetAccountRequest struct {
	ID string `json:"account_id"`
}

// GetAccountResponse represents the proto GetAccountResponse message (flat, matching gateway).
type GetAccountResponse = AccountMsg

// FreezeAccountRequest represents the proto FreezeAccountRequest message.
type FreezeAccountRequest struct {
	ID     string `json:"account_id"`
	Reason string `json:"reason"`
}

// FreezeAccountResponse represents the proto FreezeAccountResponse message (flat, matching gateway).
type FreezeAccountResponse = AccountMsg

// CloseAccountRequest represents the proto CloseAccountRequest message.
type CloseAccountRequest struct {
	ID     string `json:"account_id"`
	Reason string `json:"reason"`
}

// CloseAccountResponse represents the proto CloseAccountResponse message (flat, matching gateway).
type CloseAccountResponse = AccountMsg

// ListAccountsRequest represents the proto ListAccountsRequest message.
type ListAccountsRequest struct {
	TenantID  string `json:"tenant_id"`
	HolderID  string `json:"holder_id"`
	PageToken string `json:"page_token"`
	PageSize  int32  `json:"page_size"`
}

// ListAccountsResponse represents the proto ListAccountsResponse message.
type ListAccountsResponse struct {
	Accounts   []*AccountMsg `json:"accounts"`
	TotalCount int32         `json:"total_count"`
}

// AccountMsg represents the proto Account message.
type AccountMsg struct {
	AccountID         string `json:"account_id"`
	TenantID          string `json:"tenant_id"`
	AccountNumber     string `json:"account_number"`
	AccountType       string `json:"account_type"`
	Status            string `json:"status"`
	Currency          string `json:"currency"`
	LedgerAccountCode string `json:"ledger_account_code"`
	HolderFirstName   string `json:"holder_first_name"`
	HolderLastName    string `json:"holder_last_name"`
	HolderEmail       string `json:"holder_email"`
	Version           int32  `json:"version"`
}

// OpenAccount handles the gRPC OpenAccount request.
func (h *AccountHandler) OpenAccount(ctx context.Context, req *OpenAccountRequest) (*OpenAccountResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if !currencyCodeRE.MatchString(req.Currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}
	if req.HolderFirstName == "" {
		return nil, status.Error(codes.InvalidArgument, "holder_first_name is required")
	}
	if req.HolderLastName == "" {
		return nil, status.Error(codes.InvalidArgument, "holder_last_name is required")
	}
	if req.AccountType == "" {
		return nil, status.Error(codes.InvalidArgument, "account_type is required")
	}

	var identityVerificationID uuid.UUID
	if req.IdentityVerificationID != "" {
		identityVerificationID, err = uuid.Parse(req.IdentityVerificationID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid identity_verification_id: %v", err))
		}
	}

	result, err := h.openAccount.Execute(ctx, dto.OpenAccountRequest{
		TenantID:               tenantID,
		AccountType:            req.AccountType,
		Currency:               req.Currency,
		HolderFirstName:        req.HolderFirstName,
		HolderLastName:         req.HolderLastName,
		HolderEmail:            req.HolderEmail,
		IdentityVerificationID: identityVerificationID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &OpenAccountResponse{
		AccountID:         result.AccountID.String(),
		AccountNumber:     result.AccountNumber,
		Status:            result.Status,
		LedgerAccountCode: result.LedgerAccountCode,
	}, nil
}

// GetAccount handles the gRPC GetAccount request.
func (h *AccountHandler) GetAccount(ctx context.Context, req *GetAccountRequest) (*GetAccountResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid id: %v", err))
	}

	result, err := h.getAccount.Execute(ctx, dto.GetAccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return toAccountMsg(result), nil
}

// FreezeAccount handles the gRPC FreezeAccount request.
func (h *AccountHandler) FreezeAccount(ctx context.Context, req *FreezeAccountRequest) (*FreezeAccountResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid id: %v", err))
	}

	result, err := h.freezeAccount.Execute(ctx, dto.FreezeAccountRequest{
		AccountID: accountID,
		Reason:    req.Reason,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return toAccountMsg(result), nil
}

// CloseAccount handles the gRPC CloseAccount request.
func (h *AccountHandler) CloseAccount(ctx context.Context, req *CloseAccountRequest) (*CloseAccountResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid id: %v", err))
	}

	result, err := h.closeAccount.Execute(ctx, dto.CloseAccountRequest{
		AccountID: accountID,
		Reason:    req.Reason,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return toAccountMsg(result), nil
}

// ListAccounts handles the gRPC ListAccounts request.
func (h *AccountHandler) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*ListAccountsResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize < 0 || pageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var holderID uuid.UUID
	if req.HolderID != "" {
		holderID, err = uuid.Parse(req.HolderID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid holder_id: %v", err))
		}
	}

	result, err := h.listAccounts.Execute(ctx, dto.ListAccountsRequest{
		TenantID: tenantID,
		HolderID: holderID,
		Limit:    int(pageSize),
		Offset:   0,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	accounts := make([]*AccountMsg, 0, len(result.Accounts))
	for _, a := range result.Accounts {
		accounts = append(accounts, toAccountMsg(a))
	}

	return &ListAccountsResponse{
		Accounts:   accounts,
		TotalCount: int32(result.TotalCount), //nolint:gosec // bounded by DB query limits
	}, nil
}

func toAccountMsg(a dto.AccountResponse) *AccountMsg {
	return &AccountMsg{
		AccountID:         a.AccountID.String(),
		TenantID:          a.TenantID.String(),
		AccountNumber:     a.AccountNumber,
		AccountType:       a.AccountType,
		Status:            a.Status,
		Currency:          a.Currency,
		LedgerAccountCode: a.LedgerAccountCode,
		HolderFirstName:   a.HolderFirstName,
		HolderLastName:    a.HolderLastName,
		HolderEmail:       a.HolderEmail,
		Version:           int32(a.Version), //nolint:gosec // bounded by DB query limits
	}
}
