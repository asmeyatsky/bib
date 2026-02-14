package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// Compile-time interface check
var _ port.VerificationProvider = (*PersonaStub)(nil)

// PersonaStub is a stub implementation of the Persona KYC/AML provider.
// It returns successful results for all checks in development/test environments.
type PersonaStub struct{}

func NewPersonaStub() *PersonaStub {
	return &PersonaStub{}
}

// InitiateCheck starts a check and returns a synthetic provider reference.
func (p *PersonaStub) InitiateCheck(_ context.Context, checkType valueobject.CheckType, applicant port.ApplicantInfo) (string, error) {
	if applicant.Email == "" {
		return "", fmt.Errorf("applicant email is required")
	}

	// Generate a synthetic provider reference
	ref := fmt.Sprintf("persona-%s-%s", checkType.String(), uuid.New().String()[:8])
	return ref, nil
}

// GetCheckResult returns a successful result for stub checks.
func (p *PersonaStub) GetCheckResult(_ context.Context, providerRef string) (valueobject.VerificationStatus, string, error) {
	if providerRef == "" {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("provider reference is required")
	}

	// Stub always returns APPROVED with no failure reason
	return valueobject.StatusApproved, "", nil
}
