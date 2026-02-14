package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
)

// AssessmentRepository defines the persistence port for transaction assessments.
type AssessmentRepository interface {
	// Save persists a new or updated transaction assessment.
	Save(ctx context.Context, assessment *model.TransactionAssessment) error

	// FindByID retrieves an assessment by its unique identifier.
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error)

	// FindByTransactionID retrieves an assessment by the original transaction ID.
	FindByTransactionID(ctx context.Context, tenantID, transactionID uuid.UUID) (*model.TransactionAssessment, error)

	// FindByAccountID retrieves all assessments for a given account.
	FindByAccountID(ctx context.Context, tenantID, accountID uuid.UUID, limit, offset int) ([]*model.TransactionAssessment, error)
}

// EventPublisher defines the port for publishing domain events.
type EventPublisher interface {
	// Publish sends one or more domain events to the messaging infrastructure.
	Publish(ctx context.Context, events ...interface{}) error
}

// MLModelClient defines the port for integrating with an external ML model
// for AI-powered risk scoring (future integration).
type MLModelClient interface {
	// Predict sends feature data to an ML model and returns a risk score.
	Predict(ctx context.Context, features map[string]interface{}) (score float64, err error)
}
