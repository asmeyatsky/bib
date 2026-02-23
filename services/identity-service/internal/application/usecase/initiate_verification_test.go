package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/application/usecase"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// --- Mock implementations ---

// mockVerificationRepository implements port.VerificationRepository for testing.
type mockVerificationRepository struct {
	savedVerifications []model.IdentityVerification
	findByIDFunc       func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error)
	saveFunc           func(ctx context.Context, v model.IdentityVerification) error
}

func (m *mockVerificationRepository) Save(ctx context.Context, v model.IdentityVerification) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, v)
	}
	m.savedVerifications = append(m.savedVerifications, v)
	return nil
}

func (m *mockVerificationRepository) FindByID(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.IdentityVerification{}, fmt.Errorf("verification not found: %s", id)
}

func (m *mockVerificationRepository) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]model.IdentityVerification, int, error) {
	return nil, 0, nil
}

// mockVerificationProvider implements port.VerificationProvider for testing.
type mockVerificationProvider struct {
	initiatedChecks    []initiatedCheck
	initiateCheckFunc  func(ctx context.Context, checkType valueobject.CheckType, applicant port.ApplicantInfo) (string, error)
	getCheckResultFunc func(ctx context.Context, providerRef string) (valueobject.VerificationStatus, string, error)
}

type initiatedCheck struct {
	CheckType valueobject.CheckType
	Applicant port.ApplicantInfo
	Reference string
}

func (m *mockVerificationProvider) InitiateCheck(ctx context.Context, checkType valueobject.CheckType, applicant port.ApplicantInfo) (string, error) {
	if m.initiateCheckFunc != nil {
		return m.initiateCheckFunc(ctx, checkType, applicant)
	}
	ref := fmt.Sprintf("mock-%s-%s", checkType.String(), uuid.New().String()[:8])
	m.initiatedChecks = append(m.initiatedChecks, initiatedCheck{
		CheckType: checkType,
		Applicant: applicant,
		Reference: ref,
	})
	return ref, nil
}

func (m *mockVerificationProvider) GetCheckResult(ctx context.Context, providerRef string) (valueobject.VerificationStatus, string, error) {
	if m.getCheckResultFunc != nil {
		return m.getCheckResultFunc(ctx, providerRef)
	}
	return valueobject.StatusApproved, "", nil
}

// mockEventPublisher implements port.EventPublisher for testing.
type mockEventPublisher struct {
	publishedEvents []events.DomainEvent
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

// --- Tests ---

func validInitiateRequest() dto.InitiateVerificationRequest {
	return dto.InitiateVerificationRequest{
		TenantID:    uuid.New(),
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john.doe@example.com",
		DateOfBirth: "1990-01-15",
		Country:     "US",
	}
}

func TestInitiateVerification_Success(t *testing.T) {
	repo := &mockVerificationRepository{}
	provider := &mockVerificationProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, req.TenantID, resp.TenantID)
	assert.Equal(t, "John", resp.ApplicantFirstName)
	assert.Equal(t, "Doe", resp.ApplicantLastName)
	assert.Equal(t, "john.doe@example.com", resp.ApplicantEmail)
	assert.Equal(t, "1990-01-15", resp.ApplicantDOB)
	assert.Equal(t, "US", resp.ApplicantCountry)
	assert.Equal(t, "IN_PROGRESS", resp.Status)
	assert.Len(t, resp.Checks, 3)

	// All checks should have providers assigned and be IN_PROGRESS
	for _, c := range resp.Checks {
		assert.Equal(t, "persona", c.Provider)
		assert.NotEmpty(t, c.ProviderReference)
		assert.Equal(t, "IN_PROGRESS", c.Status)
	}

	// Verify repo was called
	require.Len(t, repo.savedVerifications, 1)

	// Verify provider was called for each check
	require.Len(t, provider.initiatedChecks, 3)
	assert.Equal(t, "John", provider.initiatedChecks[0].Applicant.FirstName)

	// Verify events were published
	assert.NotEmpty(t, publisher.publishedEvents)
}

func TestInitiateVerification_MissingFirstName(t *testing.T) {
	repo := &mockVerificationRepository{}
	provider := &mockVerificationProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	req.FirstName = ""

	_, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applicant first name is required")
	assert.Empty(t, repo.savedVerifications)
}

func TestInitiateVerification_ProviderError(t *testing.T) {
	repo := &mockVerificationRepository{}
	provider := &mockVerificationProvider{
		initiateCheckFunc: func(_ context.Context, _ valueobject.CheckType, _ port.ApplicantInfo) (string, error) {
			return "", fmt.Errorf("provider unavailable")
		},
	}
	publisher := &mockEventPublisher{}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initiate")
	assert.Contains(t, err.Error(), "provider unavailable")
	assert.Empty(t, repo.savedVerifications)
}

func TestInitiateVerification_RepoSaveError(t *testing.T) {
	repo := &mockVerificationRepository{
		saveFunc: func(_ context.Context, _ model.IdentityVerification) error {
			return fmt.Errorf("database connection lost")
		},
	}
	provider := &mockVerificationProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save verification")
	assert.Contains(t, err.Error(), "database connection lost")
	assert.Empty(t, publisher.publishedEvents)
}

func TestInitiateVerification_PublishError(t *testing.T) {
	repo := &mockVerificationRepository{}
	provider := &mockVerificationProvider{}
	publisher := &mockEventPublisher{
		publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
			return fmt.Errorf("broker unreachable")
		},
	}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish events")
	assert.Contains(t, err.Error(), "broker unreachable")
}

func TestInitiateVerification_CheckTypes(t *testing.T) {
	repo := &mockVerificationRepository{}
	provider := &mockVerificationProvider{}
	publisher := &mockEventPublisher{}

	uc := usecase.NewInitiateVerification(repo, provider, publisher)

	req := validInitiateRequest()
	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)

	// Verify the default check types are present
	checkTypes := make(map[string]bool)
	for _, c := range resp.Checks {
		checkTypes[c.CheckType] = true
	}
	assert.True(t, checkTypes["DOCUMENT"])
	assert.True(t, checkTypes["SELFIE"])
	assert.True(t, checkTypes["WATCHLIST"])
}
