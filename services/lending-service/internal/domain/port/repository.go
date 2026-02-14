package port

import (
	"context"

	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
)

// ---------------------------------------------------------------------------
// Repository ports (driven/secondary adapters)
// ---------------------------------------------------------------------------

// LoanApplicationRepository persists and retrieves loan applications.
type LoanApplicationRepository interface {
	Save(ctx context.Context, app model.LoanApplication) error
	FindByID(ctx context.Context, tenantID, id string) (model.LoanApplication, error)
	FindByApplicantID(ctx context.Context, tenantID, applicantID string) ([]model.LoanApplication, error)
}

// LoanRepository persists and retrieves loans.
type LoanRepository interface {
	Save(ctx context.Context, loan model.Loan) error
	FindByID(ctx context.Context, tenantID, id string) (model.Loan, error)
	FindByApplicationID(ctx context.Context, tenantID, applicationID string) (model.Loan, error)
	FindByBorrowerAccountID(ctx context.Context, tenantID, borrowerAccountID string) ([]model.Loan, error)
}

// CollectionCaseRepository persists and retrieves collection cases.
type CollectionCaseRepository interface {
	Save(ctx context.Context, c model.CollectionCase) error
	FindByID(ctx context.Context, tenantID, id string) (model.CollectionCase, error)
	FindByLoanID(ctx context.Context, tenantID, loanID string) ([]model.CollectionCase, error)
}

// ---------------------------------------------------------------------------
// Event publisher port
// ---------------------------------------------------------------------------

// EventPublisher publishes domain events to external consumers.
type EventPublisher interface {
	Publish(ctx context.Context, events ...event.DomainEvent) error
}

// ---------------------------------------------------------------------------
// External service ports
// ---------------------------------------------------------------------------

// CreditBureauClient fetches credit scores from an external bureau.
type CreditBureauClient interface {
	GetCreditScore(ctx context.Context, applicantID string) (string, error)
}
