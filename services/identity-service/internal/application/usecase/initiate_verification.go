package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
)

const TopicIdentityVerifications = "bib.identity.verifications"

// InitiateVerification handles the creation of a new identity verification
// and initiates checks via the external provider.
type InitiateVerification struct {
	repo     port.VerificationRepository
	provider port.VerificationProvider
	publisher port.EventPublisher
}

func NewInitiateVerification(
	repo port.VerificationRepository,
	provider port.VerificationProvider,
	publisher port.EventPublisher,
) *InitiateVerification {
	return &InitiateVerification{
		repo:      repo,
		provider:  provider,
		publisher: publisher,
	}
}

func (uc *InitiateVerification) Execute(ctx context.Context, req dto.InitiateVerificationRequest) (dto.VerificationResponse, error) {
	// Create the verification aggregate
	verification, err := model.NewIdentityVerification(
		req.TenantID,
		req.FirstName,
		req.LastName,
		req.Email,
		req.DateOfBirth,
		req.Country,
	)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to create verification: %w", err)
	}

	// Initiate checks via the external provider
	applicant := port.ApplicantInfo{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		DateOfBirth: req.DateOfBirth,
		Country:     req.Country,
	}

	for _, check := range verification.Checks() {
		providerRef, err := uc.provider.InitiateCheck(ctx, check.CheckType(), applicant)
		if err != nil {
			return dto.VerificationResponse{}, fmt.Errorf("failed to initiate %s check: %w", check.CheckType().String(), err)
		}
		verification, err = verification.UpdateCheckProvider(check.ID(), "persona", providerRef)
		if err != nil {
			return dto.VerificationResponse{}, fmt.Errorf("failed to update check provider: %w", err)
		}
	}

	// Transition to IN_PROGRESS
	now := verification.CreatedAt()
	verification, err = verification.StartProcessing(now)
	if err != nil {
		return dto.VerificationResponse{}, fmt.Errorf("failed to start processing: %w", err)
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
