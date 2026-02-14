package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
)

// AccountHandler implements the gRPC account service handler.
type AccountHandler struct {
	openAccount  *usecase.OpenAccountUseCase
	getAccount   *usecase.GetAccountUseCase
	freezeAccount *usecase.FreezeAccountUseCase
	closeAccount *usecase.CloseAccountUseCase
	listAccounts *usecase.ListAccountsUseCase
}

// NewAccountHandler creates a new gRPC account handler.
func NewAccountHandler(
	openAccount *usecase.OpenAccountUseCase,
	getAccount *usecase.GetAccountUseCase,
	freezeAccount *usecase.FreezeAccountUseCase,
	closeAccount *usecase.CloseAccountUseCase,
	listAccounts *usecase.ListAccountsUseCase,
) *AccountHandler {
	return &AccountHandler{
		openAccount:   openAccount,
		getAccount:    getAccount,
		freezeAccount: freezeAccount,
		closeAccount:  closeAccount,
		listAccounts:  listAccounts,
	}
}

// OpenAccountRequest represents the gRPC request for opening an account.
type OpenAccountRequest struct {
	TenantID               string `json:"tenant_id"`
	AccountType            string `json:"account_type"`
	Currency               string `json:"currency"`
	HolderFirstName        string `json:"holder_first_name"`
	HolderLastName         string `json:"holder_last_name"`
	HolderEmail            string `json:"holder_email"`
	IdentityVerificationID string `json:"identity_verification_id"`
}

// OpenAccountResponse represents the gRPC response for opening an account.
type OpenAccountResponse struct {
	AccountID         string `json:"account_id"`
	AccountNumber     string `json:"account_number"`
	Status            string `json:"status"`
	LedgerAccountCode string `json:"ledger_account_code"`
}

// GetAccountRequest represents the gRPC request for getting an account.
type GetAccountRequest struct {
	AccountID string `json:"account_id"`
}

// FreezeAccountRequest represents the gRPC request for freezing an account.
type FreezeAccountRequest struct {
	AccountID string `json:"account_id"`
	Reason    string `json:"reason"`
}

// CloseAccountRequest represents the gRPC request for closing an account.
type CloseAccountRequest struct {
	AccountID string `json:"account_id"`
	Reason    string `json:"reason"`
}

// ListAccountsRequest represents the gRPC request for listing accounts.
type ListAccountsRequest struct {
	TenantID string `json:"tenant_id"`
	HolderID string `json:"holder_id"`
	Limit    int32  `json:"limit"`
	Offset   int32  `json:"offset"`
}

// AccountResponse represents the gRPC response for an account.
type AccountResponse struct {
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

// ListAccountsResponse represents the gRPC response for listing accounts.
type ListAccountsResponse struct {
	Accounts   []*AccountResponse `json:"accounts"`
	TotalCount int32              `json:"total_count"`
}

// OpenAccount handles the gRPC OpenAccount request.
func (h *AccountHandler) OpenAccount(ctx context.Context, req *OpenAccountRequest) (*OpenAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid tenant_id: %v", err))
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &OpenAccountResponse{
		AccountID:         result.AccountID.String(),
		AccountNumber:     result.AccountNumber,
		Status:            result.Status,
		LedgerAccountCode: result.LedgerAccountCode,
	}, nil
}

// GetAccount handles the gRPC GetAccount request.
func (h *AccountHandler) GetAccount(ctx context.Context, req *GetAccountRequest) (*AccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid account_id: %v", err))
	}

	result, err := h.getAccount.Execute(ctx, dto.GetAccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return toAccountResponse(result), nil
}

// FreezeAccount handles the gRPC FreezeAccount request.
func (h *AccountHandler) FreezeAccount(ctx context.Context, req *FreezeAccountRequest) (*AccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid account_id: %v", err))
	}

	result, err := h.freezeAccount.Execute(ctx, dto.FreezeAccountRequest{
		AccountID: accountID,
		Reason:    req.Reason,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toAccountResponse(result), nil
}

// CloseAccount handles the gRPC CloseAccount request.
func (h *AccountHandler) CloseAccount(ctx context.Context, req *CloseAccountRequest) (*AccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid account_id: %v", err))
	}

	result, err := h.closeAccount.Execute(ctx, dto.CloseAccountRequest{
		AccountID: accountID,
		Reason:    req.Reason,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toAccountResponse(result), nil
}

// ListAccounts handles the gRPC ListAccounts request.
func (h *AccountHandler) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*ListAccountsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	var tenantID, holderID uuid.UUID
	var err error

	if req.TenantID != "" {
		tenantID, err = uuid.Parse(req.TenantID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid tenant_id: %v", err))
		}
	}

	if req.HolderID != "" {
		holderID, err = uuid.Parse(req.HolderID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid holder_id: %v", err))
		}
	}

	result, err := h.listAccounts.Execute(ctx, dto.ListAccountsRequest{
		TenantID: tenantID,
		HolderID: holderID,
		Limit:    int(req.Limit),
		Offset:   int(req.Offset),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	accounts := make([]*AccountResponse, 0, len(result.Accounts))
	for _, a := range result.Accounts {
		accounts = append(accounts, toAccountResponse(a))
	}

	return &ListAccountsResponse{
		Accounts:   accounts,
		TotalCount: int32(result.TotalCount),
	}, nil
}

func toAccountResponse(a dto.AccountResponse) *AccountResponse {
	return &AccountResponse{
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
		Version:           int32(a.Version),
	}
}
