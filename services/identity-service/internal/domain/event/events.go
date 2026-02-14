package event

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypeIdentityVerification = "IdentityVerification"

// VerificationInitiated is emitted when a new identity verification is created.
type VerificationInitiated struct {
	events.BaseEvent
	VerificationID uuid.UUID `json:"verification_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ApplicantEmail string    `json:"applicant_email"`
}

func NewVerificationInitiated(verificationID, tenantID uuid.UUID, email string) VerificationInitiated {
	payload, _ := json.Marshal(struct {
		VerificationID uuid.UUID `json:"verification_id"`
		TenantID       uuid.UUID `json:"tenant_id"`
		ApplicantEmail string    `json:"applicant_email"`
	}{verificationID, tenantID, email})

	return VerificationInitiated{
		BaseEvent:      events.NewBaseEvent("identity.verification.initiated", verificationID, AggregateTypeIdentityVerification, payload),
		VerificationID: verificationID,
		TenantID:       tenantID,
		ApplicantEmail: email,
	}
}

// VerificationCompleted is emitted when all checks pass and the verification is approved.
type VerificationCompleted struct {
	events.BaseEvent
	VerificationID uuid.UUID `json:"verification_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ApplicantEmail string    `json:"applicant_email"`
}

func NewVerificationCompleted(verificationID, tenantID uuid.UUID, email string) VerificationCompleted {
	payload, _ := json.Marshal(struct {
		VerificationID uuid.UUID `json:"verification_id"`
		TenantID       uuid.UUID `json:"tenant_id"`
		ApplicantEmail string    `json:"applicant_email"`
	}{verificationID, tenantID, email})

	return VerificationCompleted{
		BaseEvent:      events.NewBaseEvent("identity.verification.completed", verificationID, AggregateTypeIdentityVerification, payload),
		VerificationID: verificationID,
		TenantID:       tenantID,
		ApplicantEmail: email,
	}
}

// VerificationRejected is emitted when one or more checks fail and the verification is rejected.
type VerificationRejected struct {
	events.BaseEvent
	VerificationID uuid.UUID `json:"verification_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ApplicantEmail string    `json:"applicant_email"`
}

func NewVerificationRejected(verificationID, tenantID uuid.UUID, email string) VerificationRejected {
	payload, _ := json.Marshal(struct {
		VerificationID uuid.UUID `json:"verification_id"`
		TenantID       uuid.UUID `json:"tenant_id"`
		ApplicantEmail string    `json:"applicant_email"`
	}{verificationID, tenantID, email})

	return VerificationRejected{
		BaseEvent:      events.NewBaseEvent("identity.verification.rejected", verificationID, AggregateTypeIdentityVerification, payload),
		VerificationID: verificationID,
		TenantID:       tenantID,
		ApplicantEmail: email,
	}
}
