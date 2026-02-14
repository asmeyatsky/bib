package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// CompleteCheck handles webhook callbacks from the verification provider
// to mark individual checks as complete.
type CompleteCheck struct {
	repo      port.VerificationRepository
	publisher port.EventPublisher
}

func NewCompleteCheck(
	repo port.VerificationRepository,
	publisher port.EventPublisher,
) *CompleteCheck {
	return &CompleteCheck{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *CompleteCheck) Execute(ctx context.Context, req dto.CompleteCheckRequest) (dto.VerificationResponse, error) {
	// Parse status
	status, err := valueobject.NewVerificationStatus(req.Status)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("invalid status: %w", err)
	}

	// Fetch the verification
	verification, err := uc.repo.FindByID(ctx, req.VerificationID)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to find verification: %w", err)
	}

	// Complete the check
	now := time.Now().UTC()
	verification, err = verification.CompleteCheck(req.CheckID, status, req.FailureReason, now)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to complete check: %w", err)
	}

	// Persist
	if err := uc.repo.Save(ctx, verification); err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to save verification: %w", err)
	}

	// Publish domain events
	if events := verification.DomainEvents(); len(events) > 0 {
		if err := uc.publisher.Publish(ctx, TopicIdentityVerifications, events...); err != nil {
			return dto.VerificationResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	return toVerificationResponse(verification), nil
}
