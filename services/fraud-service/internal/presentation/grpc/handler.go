package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
)

// FraudServiceHandler handles gRPC requests for the fraud service.
type FraudServiceHandler struct {
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

// AssessTransactionRequest represents the gRPC request for assessing a transaction.
type AssessTransactionRequest struct {
	TenantID        string            `json:"tenant_id"`
	TransactionID   string            `json:"transaction_id"`
	AccountID       string            `json:"account_id"`
	Amount          string            `json:"amount"`
	Currency        string            `json:"currency"`
	TransactionType string            `json:"transaction_type"`
	Metadata        map[string]string `json:"metadata"`
}

// AssessTransactionResponse represents the gRPC response after assessment.
type AssessTransactionResponse struct {
	AssessmentID string   `json:"assessment_id"`
	RiskScore    int      `json:"risk_score"`
	RiskLevel    string   `json:"risk_level"`
	Decision     string   `json:"decision"`
	Signals      []string `json:"signals"`
}

// GetAssessmentRequest represents the gRPC request for retrieving an assessment.
type GetAssessmentRequest struct {
	TenantID     string `json:"tenant_id"`
	AssessmentID string `json:"assessment_id"`
}

// GetAssessmentResponse represents the gRPC response with assessment details.
type GetAssessmentResponse struct {
	AssessmentID    string   `json:"assessment_id"`
	TransactionID   string   `json:"transaction_id"`
	AccountID       string   `json:"account_id"`
	Amount          string   `json:"amount"`
	Currency        string   `json:"currency"`
	TransactionType string   `json:"transaction_type"`
	RiskScore       int      `json:"risk_score"`
	RiskLevel       string   `json:"risk_level"`
	Decision        string   `json:"decision"`
	Signals         []string `json:"signals"`
}

// AssessTransaction handles a transaction assessment request.
func (h *FraudServiceHandler) AssessTransaction(ctx context.Context, req *AssessTransactionRequest) (*AssessTransactionResponse, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	transactionID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction_id: %w", err)
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
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
		Currency:        req.Currency,
		TransactionType: req.TransactionType,
		Metadata:        req.Metadata,
	})
	if err != nil {
		h.logger.Error("failed to assess transaction",
			slog.String("transaction_id", transactionID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("assessment failed: %w", err)
	}

	return &AssessTransactionResponse{
		AssessmentID: result.ID.String(),
		RiskScore:    result.RiskScore,
		RiskLevel:    result.RiskLevel,
		Decision:     result.Decision,
		Signals:      result.RiskSignals,
	}, nil
}

// GetAssessment handles a get assessment request.
func (h *FraudServiceHandler) GetAssessment(ctx context.Context, req *GetAssessmentRequest) (*GetAssessmentResponse, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	assessmentID, err := uuid.Parse(req.AssessmentID)
	if err != nil {
		return nil, fmt.Errorf("invalid assessment_id: %w", err)
	}

	result, err := h.getAssessment.Execute(ctx, dto.GetAssessmentRequest{
		TenantID:     tenantID,
		AssessmentID: assessmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get assessment: %w", err)
	}

	return &GetAssessmentResponse{
		AssessmentID:    result.ID.String(),
		TransactionID:   result.TransactionID.String(),
		AccountID:       result.AccountID.String(),
		Amount:          result.Amount,
		Currency:        result.Currency,
		TransactionType: result.TransactionType,
		RiskScore:       result.RiskScore,
		RiskLevel:       result.RiskLevel,
		Decision:        result.Decision,
		Signals:         result.RiskSignals,
	}, nil
}
