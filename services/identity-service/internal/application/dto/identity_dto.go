package dto

import (
	"time"

	"github.com/google/uuid"
)

// InitiateVerificationRequest is the input DTO for initiating a new verification.
type InitiateVerificationRequest struct {
	FirstName   string
	LastName    string
	Email       string
	DateOfBirth string
	Country     string
	TenantID    uuid.UUID
}

// GetVerificationRequest is the input DTO for retrieving a verification.
type GetVerificationRequest struct {
	ID uuid.UUID
}

// CompleteCheckRequest is the input DTO for completing a verification check (webhook callback).
type CompleteCheckRequest struct {
	Status         string
	FailureReason  string
	VerificationID uuid.UUID
	CheckID        uuid.UUID
}

// ListVerificationsRequest is the input DTO for listing verifications by tenant.
type ListVerificationsRequest struct {
	TenantID uuid.UUID
	PageSize int
	Offset   int
}

// VerificationCheckDTO transfers check data across layer boundaries.
type VerificationCheckDTO struct {
	CompletedAt       *time.Time
	CheckType         string
	Status            string
	Provider          string
	ProviderReference string
	FailureReason     string
	ID                uuid.UUID
}

// VerificationResponse is the output DTO for a verification.
type VerificationResponse struct {
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ApplicantFirstName string
	ApplicantLastName  string
	ApplicantEmail     string
	ApplicantDOB       string
	ApplicantCountry   string
	Status             string
	Checks             []VerificationCheckDTO
	Version            int
	ID                 uuid.UUID
	TenantID           uuid.UUID
}

// ListVerificationsResponse is the output DTO for listing verifications.
type ListVerificationsResponse struct {
	Verifications []VerificationResponse
	TotalCount    int
}
