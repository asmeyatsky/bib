package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/identity-service/internal/domain/event"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// IdentityVerification is the root aggregate for the identity bounded context.
// It orchestrates KYC/AML verification for an applicant.
type IdentityVerification struct {
	id                 uuid.UUID
	tenantID           uuid.UUID
	applicantFirstName string
	applicantLastName  string
	applicantEmail     string
	applicantDOB       string
	applicantCountry   string
	status             valueobject.VerificationStatus
	checks             []VerificationCheck
	version            int
	createdAt          time.Time
	updatedAt          time.Time
	domainEvents       []events.DomainEvent
}

// NewIdentityVerification creates a new verification in PENDING status
// with the default set of checks (DOCUMENT, SELFIE, WATCHLIST).
func NewIdentityVerification(
	tenantID uuid.UUID,
	firstName, lastName, email, dob, country string,
) (IdentityVerification, error) {
	if tenantID == uuid.Nil {
		return IdentityVerification{}, fmt.Errorf("tenant ID is required")
	}
	if firstName == "" {
		return IdentityVerification{}, fmt.Errorf("applicant first name is required")
	}
	if lastName == "" {
		return IdentityVerification{}, fmt.Errorf("applicant last name is required")
	}
	if email == "" {
		return IdentityVerification{}, fmt.Errorf("applicant email is required")
	}
	if dob == "" {
		return IdentityVerification{}, fmt.Errorf("applicant date of birth is required")
	}
	if country == "" {
		return IdentityVerification{}, fmt.Errorf("applicant country is required")
	}

	id := uuid.New()
	now := time.Now().UTC()

	// Create default checks
	var checks []VerificationCheck
	for _, ct := range valueobject.DefaultCheckTypes() {
		checks = append(checks, NewVerificationCheck(ct))
	}

	v := IdentityVerification{
		id:                 id,
		tenantID:           tenantID,
		applicantFirstName: firstName,
		applicantLastName:  lastName,
		applicantEmail:     email,
		applicantDOB:       dob,
		applicantCountry:   country,
		status:             valueobject.StatusPending,
		checks:             checks,
		version:            1,
		createdAt:          now,
		updatedAt:          now,
	}

	v.domainEvents = append(v.domainEvents, event.NewVerificationInitiated(id, tenantID, email))

	return v, nil
}

// Reconstruct recreates an IdentityVerification from persistence (no validation, no events).
func Reconstruct(
	id, tenantID uuid.UUID,
	firstName, lastName, email, dob, country string,
	status valueobject.VerificationStatus,
	checks []VerificationCheck,
	version int,
	createdAt, updatedAt time.Time,
) IdentityVerification {
	return IdentityVerification{
		id:                 id,
		tenantID:           tenantID,
		applicantFirstName: firstName,
		applicantLastName:  lastName,
		applicantEmail:     email,
		applicantDOB:       dob,
		applicantCountry:   country,
		status:             status,
		checks:             checks,
		version:            version,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
	}
}

// StartProcessing transitions the verification from PENDING to IN_PROGRESS (immutable - returns new copy).
func (v IdentityVerification) StartProcessing(now time.Time) (IdentityVerification, error) {
	if v.status != valueobject.StatusPending {
		return IdentityVerification{}, fmt.Errorf("can only start processing verifications in PENDING status, current: %s", v.status.String())
	}

	updated := v
	updated.status = valueobject.StatusInProgress
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = copyEvents(v.domainEvents)

	// Transition all pending checks to IN_PROGRESS
	newChecks := make([]VerificationCheck, len(v.checks))
	for i, c := range v.checks {
		if c.Status().Equal(valueobject.StatusPending) {
			started, err := c.SetInProgress()
			if err != nil {
				return IdentityVerification{}, fmt.Errorf("failed to start check %s: %w", c.ID(), err)
			}
			newChecks[i] = started
		} else {
			newChecks[i] = c
		}
	}
	updated.checks = newChecks

	return updated, nil
}

// CompleteCheck updates a specific check within this verification and evaluates overall status.
// This is immutable - returns a new copy of the aggregate.
func (v IdentityVerification) CompleteCheck(
	checkID uuid.UUID,
	status valueobject.VerificationStatus,
	failureReason string,
	now time.Time,
) (IdentityVerification, error) {
	if v.status.IsTerminal() {
		return IdentityVerification{}, fmt.Errorf("verification %s is already in terminal status %s", v.id, v.status.String())
	}

	updated := v
	updated.updatedAt = now
	updated.version++
	updated.domainEvents = copyEvents(v.domainEvents)

	// Find and complete the target check
	found := false
	newChecks := make([]VerificationCheck, len(v.checks))
	for i, c := range v.checks {
		if c.ID() == checkID {
			completed, err := c.Complete(status, failureReason, now)
			if err != nil {
				return IdentityVerification{}, fmt.Errorf("failed to complete check: %w", err)
			}
			newChecks[i] = completed
			found = true
		} else {
			newChecks[i] = c
		}
	}
	if !found {
		return IdentityVerification{}, fmt.Errorf("check %s not found in verification %s", checkID, v.id)
	}
	updated.checks = newChecks

	// Evaluate overall status if all checks are now complete
	updated = updated.evaluateOverallStatus()

	return updated, nil
}

// UpdateCheckProvider sets the provider and reference on a specific check (immutable).
func (v IdentityVerification) UpdateCheckProvider(checkID uuid.UUID, provider, providerRef string) (IdentityVerification, error) {
	updated := v
	updated.domainEvents = copyEvents(v.domainEvents)

	found := false
	newChecks := make([]VerificationCheck, len(v.checks))
	for i, c := range v.checks {
		if c.ID() == checkID {
			newChecks[i] = c.SetProvider(provider, providerRef)
			found = true
		} else {
			newChecks[i] = c
		}
	}
	if !found {
		return IdentityVerification{}, fmt.Errorf("check %s not found in verification %s", checkID, v.id)
	}
	updated.checks = newChecks
	return updated, nil
}

// evaluateOverallStatus determines the aggregate status based on individual check results.
// If any check is REJECTED -> overall REJECTED.
// If all checks are APPROVED -> overall APPROVED.
// Otherwise the status remains unchanged.
func (v IdentityVerification) evaluateOverallStatus() IdentityVerification {
	allTerminal := true
	allApproved := true

	for _, c := range v.checks {
		if !c.Status().IsTerminal() {
			allTerminal = false
			allApproved = false
			break
		}
		if c.Status().Equal(valueobject.StatusRejected) {
			// At least one rejection -> overall rejected immediately
			result := v
			result.status = valueobject.StatusRejected
			result.domainEvents = append(result.domainEvents,
				event.NewVerificationRejected(v.id, v.tenantID, v.applicantEmail))
			return result
		}
		if !c.Status().Equal(valueobject.StatusApproved) {
			allApproved = false
		}
	}

	if allTerminal && allApproved {
		result := v
		result.status = valueobject.StatusApproved
		result.domainEvents = append(result.domainEvents,
			event.NewVerificationCompleted(v.id, v.tenantID, v.applicantEmail))
		return result
	}

	return v
}

// copyEvents creates a defensive copy of domain events.
func copyEvents(src []events.DomainEvent) []events.DomainEvent {
	if src == nil {
		return nil
	}
	dst := make([]events.DomainEvent, len(src))
	copy(dst, src)
	return dst
}

// Accessors

func (v IdentityVerification) ID() uuid.UUID                         { return v.id }
func (v IdentityVerification) TenantID() uuid.UUID                   { return v.tenantID }
func (v IdentityVerification) ApplicantFirstName() string            { return v.applicantFirstName }
func (v IdentityVerification) ApplicantLastName() string             { return v.applicantLastName }
func (v IdentityVerification) ApplicantEmail() string                { return v.applicantEmail }
func (v IdentityVerification) ApplicantDOB() string                  { return v.applicantDOB }
func (v IdentityVerification) ApplicantCountry() string              { return v.applicantCountry }
func (v IdentityVerification) Status() valueobject.VerificationStatus { return v.status }
func (v IdentityVerification) Version() int                          { return v.version }
func (v IdentityVerification) CreatedAt() time.Time                  { return v.createdAt }
func (v IdentityVerification) UpdatedAt() time.Time                  { return v.updatedAt }
func (v IdentityVerification) DomainEvents() []events.DomainEvent    { return v.domainEvents }

func (v IdentityVerification) Checks() []VerificationCheck {
	result := make([]VerificationCheck, len(v.checks))
	copy(result, v.checks)
	return result
}

func (v IdentityVerification) ClearDomainEvents() []events.DomainEvent {
	evts := v.domainEvents
	v.domainEvents = nil
	return evts
}
