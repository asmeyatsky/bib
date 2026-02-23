package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// VerificationCheck is a child entity within the IdentityVerification aggregate.
// It represents a single verification check (e.g. document, selfie, watchlist).
type VerificationCheck struct {
	id                uuid.UUID
	checkType         valueobject.CheckType
	status            valueobject.VerificationStatus
	provider          string
	providerReference string
	failureReason     string
	completedAt       *time.Time
}

// NewVerificationCheck creates a new check in PENDING status.
func NewVerificationCheck(checkType valueobject.CheckType) VerificationCheck {
	return VerificationCheck{
		id:        uuid.New(),
		checkType: checkType,
		status:    valueobject.StatusPending,
	}
}

// ReconstructCheck recreates a VerificationCheck from persistence (no validation, no events).
func ReconstructCheck(
	id uuid.UUID,
	checkType valueobject.CheckType,
	status valueobject.VerificationStatus,
	provider string,
	providerReference string,
	completedAt *time.Time,
	failureReason string,
) VerificationCheck {
	return VerificationCheck{
		id:                id,
		checkType:         checkType,
		status:            status,
		provider:          provider,
		providerReference: providerReference,
		completedAt:       completedAt,
		failureReason:     failureReason,
	}
}

// Complete transitions a check to a terminal status (immutable - returns new copy).
func (vc VerificationCheck) Complete(status valueobject.VerificationStatus, failureReason string, completedAt time.Time) (VerificationCheck, error) {
	if vc.status.IsTerminal() {
		return VerificationCheck{}, fmt.Errorf("check %s is already in terminal status %s", vc.id, vc.status.String())
	}
	if !status.IsTerminal() {
		return VerificationCheck{}, fmt.Errorf("completion status must be terminal, got: %s", status.String())
	}

	completed := vc
	completed.status = status
	completed.failureReason = failureReason
	completed.completedAt = &completedAt
	return completed, nil
}

// SetProvider sets the provider and reference on a check (immutable - returns new copy).
func (vc VerificationCheck) SetProvider(provider, providerRef string) VerificationCheck {
	updated := vc
	updated.provider = provider
	updated.providerReference = providerRef
	return updated
}

// SetInProgress transitions a check to IN_PROGRESS status (immutable - returns new copy).
func (vc VerificationCheck) SetInProgress() (VerificationCheck, error) {
	if vc.status != valueobject.StatusPending {
		return VerificationCheck{}, fmt.Errorf("can only start processing checks in PENDING status, current: %s", vc.status.String())
	}
	updated := vc
	updated.status = valueobject.StatusInProgress
	return updated, nil
}

// Accessors

func (vc VerificationCheck) ID() uuid.UUID                         { return vc.id }
func (vc VerificationCheck) CheckType() valueobject.CheckType      { return vc.checkType }
func (vc VerificationCheck) Status() valueobject.VerificationStatus { return vc.status }
func (vc VerificationCheck) Provider() string                      { return vc.provider }
func (vc VerificationCheck) ProviderReference() string             { return vc.providerReference }
func (vc VerificationCheck) FailureReason() string                 { return vc.failureReason }

func (vc VerificationCheck) CompletedAt() *time.Time {
	if vc.completedAt == nil {
		return nil
	}
	t := *vc.completedAt
	return &t
}
