package dto

import (
	"time"

	"github.com/google/uuid"
)

// InitiateVerificationRequest is the input DTO for initiating a new verification.
type InitiateVerificationRequest struct {
	TenantID    uuid.UUID
	FirstName   string
	LastName    string
	Email       string
	DateOfBirth string
	Country     string
}

// GetVerificationRequest is the input DTO for retrieving a verification.
type GetVerificationRequest struct {
	ID uuid.UUID
}

// CompleteCheckRequest is the input DTO for completing a verification check (webhook callback).
type CompleteCheckRequest struct {
	VerificationID uuid.UUID
	CheckID        uuid.UUID
	Status         string
	FailureReason  string
}

// ListVerificationsRequest is the input DTO for listing verifications by tenant.
type ListVerificationsRequest struct {
	TenantID uuid.UUID
	PageSize int
	Offset   int
}

// VerificationCheckDTO transfers check data across layer boundaries.
type VerificationCheckDTO struct {
	ID                uuid.UUID
	CheckType         string
	Status            string
	Provider          string
	ProviderReference string
	CompletedAt       *time.Time
	FailureReason     string
}

// VerificationResponse is the output DTO for a verification.
type VerificationResponse struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	ApplicantFirstName string
	ApplicantLastName  string
	ApplicantEmail     string
	ApplicantDOB       string
	ApplicantCountry   string
	Status             string
	Checks             []VerificationCheckDTO
	Version            int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ListVerificationsResponse is the output DTO for listing verifications.
type ListVerificationsResponse struct {
	Verifications []VerificationResponse
	TotalCount    int
}
