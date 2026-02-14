package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/lending-service/internal/application/dto"
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/port"
	"github.com/bibbank/bib/services/lending-service/internal/domain/service"
)

// SubmitLoanApplicationUseCase orchestrates new loan application submission,
// credit score fetching, and underwriting.
type SubmitLoanApplicationUseCase struct {
	appRepo      port.LoanApplicationRepository
	publisher    port.EventPublisher
	creditClient port.CreditBureauClient
	underwriter  *service.UnderwritingEngine
}

// NewSubmitLoanApplicationUseCase wires dependencies.
func NewSubmitLoanApplicationUseCase(
	appRepo port.LoanApplicationRepository,
	publisher port.EventPublisher,
	creditClient port.CreditBureauClient,
	underwriter *service.UnderwritingEngine,
) *SubmitLoanApplicationUseCase {
	return &SubmitLoanApplicationUseCase{
		appRepo:      appRepo,
		publisher:    publisher,
		creditClient: creditClient,
		underwriter:  underwriter,
	}
}

// Execute creates, underwrites, and persists a loan application.
func (uc *SubmitLoanApplicationUseCase) Execute(
	ctx context.Context,
	req dto.SubmitApplicationRequest,
) (dto.LoanApplicationResponse, error) {
	now := time.Now().UTC()

	// 1. Create the application aggregate.
	app, err := model.NewLoanApplication(
		req.TenantID, req.ApplicantID, req.RequestedAmount,
		req.Currency, req.TermMonths, req.Purpose, now,
	)
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("create application: %w", err)
	}

	// 2. Submit for review.
	app, err = app.SubmitForReview(now)
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("submit for review: %w", err)
	}

	// 3. Fetch credit score from bureau.
	creditScore, err := uc.creditClient.GetCreditScore(ctx, req.ApplicantID)
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("fetch credit score: %w", err)
	}

	// 4. Run underwriting engine.
	result := uc.underwriter.Evaluate(creditScore, req.RequestedAmount, req.TermMonths)

	// 5. Apply decision.
	if result.Approved {
		app, err = app.Approve(result.Reason, result.CreditScore, now)
	} else {
		app, err = app.Reject(result.Reason, now)
	}
	if err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("apply decision: %w", err)
	}

	// 6. Persist.
	if err := uc.appRepo.Save(ctx, app); err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("save application: %w", err)
	}

	// 7. Publish domain events.
	if err := uc.publisher.Publish(ctx, app.DomainEvents()...); err != nil {
		return dto.LoanApplicationResponse{}, fmt.Errorf("publish events: %w", err)
	}

	return toApplicationResponse(app), nil
}

func toApplicationResponse(app model.LoanApplication) dto.LoanApplicationResponse {
	return dto.LoanApplicationResponse{
		ID:              app.ID(),
		TenantID:        app.TenantID(),
		ApplicantID:     app.ApplicantID(),
		RequestedAmount: app.RequestedAmount(),
		Currency:        app.Currency(),
		TermMonths:      app.TermMonths(),
		Purpose:         app.Purpose(),
		Status:          app.Status().String(),
		DecisionReason:  app.DecisionReason(),
		CreditScore:     app.CreditScore(),
		CreatedAt:       app.CreatedAt(),
		UpdatedAt:       app.UpdatedAt(),
	}
}
