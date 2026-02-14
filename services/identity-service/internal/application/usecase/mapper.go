package usecase

import (
	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
)

// toVerificationResponse maps a domain model to a response DTO.
func toVerificationResponse(v model.IdentityVerification) dto.VerificationResponse {
	var checks []dto.VerificationCheckDTO
	for _, c := range v.Checks() {
		checks = append(checks, dto.VerificationCheckDTO{
			ID:                c.ID(),
			CheckType:         c.CheckType().String(),
			Status:            c.Status().String(),
			Provider:          c.Provider(),
			ProviderReference: c.ProviderReference(),
			CompletedAt:       c.CompletedAt(),
			FailureReason:     c.FailureReason(),
		})
	}

	return dto.VerificationResponse{
		ID:                 v.ID(),
		TenantID:           v.TenantID(),
		ApplicantFirstName: v.ApplicantFirstName(),
		ApplicantLastName:  v.ApplicantLastName(),
		ApplicantEmail:     v.ApplicantEmail(),
		ApplicantDOB:       v.ApplicantDOB(),
		ApplicantCountry:   v.ApplicantCountry(),
		Status:             v.Status().String(),
		Checks:             checks,
		Version:            v.Version(),
		CreatedAt:          v.CreatedAt(),
		UpdatedAt:          v.UpdatedAt(),
	}
}
