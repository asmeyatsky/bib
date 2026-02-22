package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
)

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

// Compile-time assertion that FraudServiceHandler implements FraudServiceServer.
var _ FraudServiceServer = (*FraudServiceHandler)(nil)

// FraudServiceHandler implements the gRPC FraudServiceServer interface.
type FraudServiceHandler struct {
	UnimplementedFraudServiceServer
	assessTransaction *usecase.AssessTransaction
	getAssessment     *usecase.GetAssessment
	logger            *slog.Logger
}

// NewFraudServiceHandler creates a new gRPC handler.
func NewFraudServiceHandler(
	assessTransaction *usecase.AssessTransaction,
	getAssessment *usecase.GetAssessment,
	logger *slog.Logger,
) *FraudServiceHandler {
	return &FraudServiceHandler{
		assessTransaction: assessTransaction,
		getAssessment:     getAssessment,
		logger:            logger,
	}
}

// Proto-aligned request/response message types.

// AssessTransactionRequest represents the proto AssessTransactionRequest message.
type AssessTransactionRequest struct {
	TenantID        string            `json:"tenant_id"`
	TransactionID   string            `json:"transaction_id"`
	AccountID       string            `json:"account_id"`
	Amount          *MoneyMsg         `json:"amount"`
	TransactionType string            `json:"transaction_type"`
	Metadata        map[string]string `json:"metadata"`
}

// MoneyMsg represents the proto Money message.
type MoneyMsg struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// TransactionAssessmentMsg represents the proto TransactionAssessment message.
type TransactionAssessmentMsg struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	TransactionID   string    `json:"transaction_id"`
	AccountID       string    `json:"account_id"`
	Amount          *MoneyMsg `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	RiskLevel       string    `json:"risk_level"`
	RiskScore       int32     `json:"risk_score"`
	Decision        string    `json:"decision"`
	RiskSignals     []string  `json:"risk_signals"`
}

// AssessTransactionResponse represents the proto AssessTransactionResponse message.
type AssessTransactionResponse struct {
	Assessment *TransactionAssessmentMsg `json:"assessment"`
}

// GetAssessmentRequest represents the proto GetAssessmentRequest message.
type GetAssessmentRequest struct {
	ID string `json:"id"`
}

// GetAssessmentResponse represents the proto GetAssessmentResponse message.
type GetAssessmentResponse struct {
	Assessment *TransactionAssessmentMsg `json:"assessment"`
}

// AssessTransaction handles a transaction assessment request.
func (h *FraudServiceHandler) AssessTransaction(ctx context.Context, req *AssessTransactionRequest) (*AssessTransactionResponse, error) {
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

	transactionID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_id: %v", err)
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account_id: %v", err)
	}

	var amount decimal.Decimal
	var currency string
	if req.Amount != nil {
		amount, err = decimal.NewFromString(req.Amount.Amount)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
		}
		currency = req.Amount.Currency
	}

	h.logger.Info("assessing transaction",
		slog.String("tenant_id", tenantID.String()),
		slog.String("transaction_id", transactionID.String()),
	)

	result, err := h.assessTransaction.Execute(ctx, dto.AssessTransactionRequest{
		TenantID:        tenantID,
		TransactionID:   transactionID,
		AccountID:       accountID,
		Amount:          amount,
		Currency:        currency,
		TransactionType: req.TransactionType,
		Metadata:        req.Metadata,
	})
	if err != nil {
		h.logger.Error("failed to assess transaction",
			slog.String("transaction_id", transactionID.String()),
			slog.String("error", err.Error()),
		)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &AssessTransactionResponse{
		Assessment: &TransactionAssessmentMsg{
			ID:              result.ID.String(),
			TenantID:        tenantID.String(),
			TransactionID:   transactionID.String(),
			AccountID:       accountID.String(),
			Amount:          req.Amount,
			TransactionType: req.TransactionType,
			RiskLevel:       result.RiskLevel,
			RiskScore:       int32(result.RiskScore),
			Decision:        result.Decision,
			RiskSignals:     result.RiskSignals,
		},
	}, nil
}

// GetAssessment handles a get assessment request.
func (h *FraudServiceHandler) GetAssessment(ctx context.Context, req *GetAssessmentRequest) (*GetAssessmentResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	assessmentID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	result, err := h.getAssessment.Execute(ctx, dto.GetAssessmentRequest{
		TenantID:     tenantID,
		AssessmentID: assessmentID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetAssessmentResponse{
		Assessment: &TransactionAssessmentMsg{
			ID:              result.ID.String(),
			TransactionID:   result.TransactionID.String(),
			AccountID:       result.AccountID.String(),
			Amount:          &MoneyMsg{Amount: result.Amount, Currency: result.Currency},
			TransactionType: result.TransactionType,
			RiskLevel:       result.RiskLevel,
			RiskScore:       int32(result.RiskScore),
			Decision:        result.Decision,
			RiskSignals:     result.RiskSignals,
		},
	}, nil
}
