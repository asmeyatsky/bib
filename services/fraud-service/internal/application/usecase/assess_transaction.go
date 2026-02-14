package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/fraud-service/internal/application/dto"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/port"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
)

// AssessTransaction is the use case for scoring and assessing a transaction.
type AssessTransaction struct {
	repo      port.AssessmentRepository
	publisher port.EventPublisher
	scorer    *service.RiskScorer
}

// NewAssessTransaction creates a new AssessTransaction use case.
func NewAssessTransaction(
	repo port.AssessmentRepository,
	publisher port.EventPublisher,
	scorer *service.RiskScorer,
) *AssessTransaction {
	return &AssessTransaction{
		repo:      repo,
		publisher: publisher,
		scorer:    scorer,
	}
}

// Execute performs risk scoring, creates the assessment, persists it, and publishes events.
func (uc *AssessTransaction) Execute(ctx context.Context, req dto.AssessTransactionRequest) (dto.AssessmentResponse, error) {
	// 1. Create the assessment aggregate.
	assessment, err := model.NewTransactionAssessment(
		req.TenantID,
		req.TransactionID,
		req.AccountID,
		req.Amount,
		req.Currency,
		req.TransactionType,
	)
	if err != nil {
		return dto.AssessmentResponse{}, fmt.Errorf("failed to create assessment: %w", err)
	}

	// 2. Run risk scoring via the domain service.
	riskInput := service.RiskInput{
		Amount:          req.Amount,
		Currency:        req.Currency,
		AccountID:       req.AccountID,
		TransactionType: req.TransactionType,
		Metadata:        req.Metadata,
	}
	riskOutput := uc.scorer.Score(riskInput)

	// 3. Apply the score to the assessment (this determines risk level and decision).
	if err := assessment.Assess(riskOutput.Score, riskOutput.Signals); err != nil {
		return dto.AssessmentResponse{}, fmt.Errorf("failed to assess transaction: %w", err)
	}

	// 4. Persist the assessment.
	if err := uc.repo.Save(ctx, assessment); err != nil {
		return dto.AssessmentResponse{}, fmt.Errorf("failed to save assessment: %w", err)
	}

	// 5. Publish domain events.
	events := assessment.DomainEvents()
	if len(events) > 0 {
		if err := uc.publisher.Publish(ctx, events...); err != nil {
			return dto.AssessmentResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	return dto.FromModel(assessment), nil
}
