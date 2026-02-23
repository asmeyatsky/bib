package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
)

// AssessTransactionRequest is the input DTO for the AssessTransaction use case.
type AssessTransactionRequest struct {
	Metadata        map[string]string `json:"metadata"`
	Amount          decimal.Decimal   `json:"amount"`
	Currency        string            `json:"currency"`
	TransactionType string            `json:"transaction_type"`
	TenantID        uuid.UUID         `json:"tenant_id"`
	TransactionID   uuid.UUID         `json:"transaction_id"`
	AccountID       uuid.UUID         `json:"account_id"`
}

// AssessmentResponse is the output DTO returned after an assessment.
type AssessmentResponse struct {
	CreatedAt       time.Time `json:"created_at"`
	AssessedAt      time.Time `json:"assessed_at"`
	TransactionType string    `json:"transaction_type"`
	Decision        string    `json:"decision"`
	RiskLevel       string    `json:"risk_level"`
	Amount          string    `json:"amount"`
	Currency        string    `json:"currency"`
	RiskSignals     []string  `json:"risk_signals"`
	RiskScore       int       `json:"risk_score"`
	ID              uuid.UUID `json:"id"`
	AccountID       uuid.UUID `json:"account_id"`
	TransactionID   uuid.UUID `json:"transaction_id"`
	TenantID        uuid.UUID `json:"tenant_id"`
}

// GetAssessmentRequest is the input DTO for retrieving an assessment.
type GetAssessmentRequest struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	AssessmentID uuid.UUID `json:"assessment_id"`
}

// FromModel maps a domain model to the response DTO.
func FromModel(a *model.TransactionAssessment) AssessmentResponse {
	return AssessmentResponse{
		ID:              a.ID(),
		TenantID:        a.TenantID(),
		TransactionID:   a.TransactionID(),
		AccountID:       a.AccountID(),
		Amount:          a.Amount().String(),
		Currency:        a.Currency(),
		TransactionType: a.TransactionType(),
		RiskLevel:       a.RiskLevel().String(),
		RiskScore:       a.RiskScore(),
		Decision:        a.Decision().String(),
		RiskSignals:     a.RiskSignals(),
		AssessedAt:      a.AssessedAt(),
		CreatedAt:       a.CreatedAt(),
	}
}
