package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// VerificationRepository defines persistence operations for identity verifications.
type VerificationRepository interface {
	// Save persists an identity verification (insert or update).
	Save(ctx context.Context, v model.IdentityVerification) error
	// FindByID retrieves a verification by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error)
	// ListByTenant returns verifications for a tenant with pagination.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.IdentityVerification, int, error)
}

// ApplicantInfo holds the applicant data needed by a verification provider.
type ApplicantInfo struct {
	FirstName   string
	LastName    string
	Email       string
	DateOfBirth string
	Country     string
}

// VerificationProvider defines the interface for external KYC/AML providers.
type VerificationProvider interface {
	// InitiateCheck starts a check with the external provider and returns a provider reference.
	InitiateCheck(ctx context.Context, checkType valueobject.CheckType, applicant ApplicantInfo) (providerRef string, err error)
	// GetCheckResult retrieves the result of a previously initiated check.
	GetCheckResult(ctx context.Context, providerRef string) (valueobject.VerificationStatus, string, error)
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...events.DomainEvent) error
}
