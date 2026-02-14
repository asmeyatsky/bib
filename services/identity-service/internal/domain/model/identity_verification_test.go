package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

func TestNewIdentityVerification_Valid(t *testing.T) {
	tenantID := uuid.New()

	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, v.ID())
	assert.Equal(t, tenantID, v.TenantID())
	assert.Equal(t, "John", v.ApplicantFirstName())
	assert.Equal(t, "Doe", v.ApplicantLastName())
	assert.Equal(t, "john@example.com", v.ApplicantEmail())
	assert.Equal(t, "1990-01-15", v.ApplicantDOB())
	assert.Equal(t, "US", v.ApplicantCountry())
	assert.True(t, v.Status().Equal(valueobject.StatusPending))
	assert.Equal(t, 1, v.Version())
	assert.False(t, v.CreatedAt().IsZero())
	assert.False(t, v.UpdatedAt().IsZero())

	// Should have default checks
	checks := v.Checks()
	require.Len(t, checks, 3)
	assert.True(t, checks[0].CheckType().Equal(valueobject.CheckTypeDocument))
	assert.True(t, checks[1].CheckType().Equal(valueobject.CheckTypeSelfie))
	assert.True(t, checks[2].CheckType().Equal(valueobject.CheckTypeWatchlist))
	for _, c := range checks {
		assert.True(t, c.Status().Equal(valueobject.StatusPending))
	}

	// Should emit VerificationInitiated event
	events := v.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "identity.verification.initiated", events[0].EventType())
}

func TestNewIdentityVerification_MissingTenantID(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.Nil, "John", "Doe", "john@example.com", "1990-01-15", "US")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewIdentityVerification_MissingFirstName(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.New(), "", "Doe", "john@example.com", "1990-01-15", "US")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicant first name is required")
}

func TestNewIdentityVerification_MissingLastName(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.New(), "John", "", "john@example.com", "1990-01-15", "US")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicant last name is required")
}

func TestNewIdentityVerification_MissingEmail(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.New(), "John", "Doe", "", "1990-01-15", "US")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicant email is required")
}

func TestNewIdentityVerification_MissingDOB(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.New(), "John", "Doe", "john@example.com", "", "US")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicant date of birth is required")
}

func TestNewIdentityVerification_MissingCountry(t *testing.T) {
	_, err := model.NewIdentityVerification(uuid.New(), "John", "Doe", "john@example.com", "1990-01-15", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicant country is required")
}

func TestIdentityVerification_StartProcessing(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	processing, err := v.StartProcessing(now)
	require.NoError(t, err)

	assert.True(t, processing.Status().Equal(valueobject.StatusInProgress))
	assert.Equal(t, 2, processing.Version())
	assert.Equal(t, now, processing.UpdatedAt())

	// All checks should be IN_PROGRESS
	for _, c := range processing.Checks() {
		assert.True(t, c.Status().Equal(valueobject.StatusInProgress), "check %s should be IN_PROGRESS", c.CheckType().String())
	}

	// Original should remain unchanged (immutable)
	assert.True(t, v.Status().Equal(valueobject.StatusPending))
	assert.Equal(t, 1, v.Version())
}

func TestIdentityVerification_StartProcessing_FromNonPending_Error(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	processing, err := v.StartProcessing(now)
	require.NoError(t, err)

	// Try to start processing again
	_, err = processing.StartProcessing(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only start processing verifications in PENDING status")
}

func TestIdentityVerification_CompleteCheck_AllApproved(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	checks := v.Checks()
	require.Len(t, checks, 3)

	// Complete all checks as APPROVED
	completionTime := now.Add(time.Minute)
	for _, c := range checks {
		v, err = v.CompleteCheck(c.ID(), valueobject.StatusApproved, "", completionTime)
		require.NoError(t, err)
	}

	// Overall status should be APPROVED
	assert.True(t, v.Status().Equal(valueobject.StatusApproved))

	// All checks should be APPROVED
	for _, c := range v.Checks() {
		assert.True(t, c.Status().Equal(valueobject.StatusApproved))
		assert.NotNil(t, c.CompletedAt())
		assert.Empty(t, c.FailureReason())
	}

	// Should have emitted VerificationCompleted event
	events := v.DomainEvents()
	found := false
	for _, e := range events {
		if e.EventType() == "identity.verification.completed" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have emitted VerificationCompleted event")
}

func TestIdentityVerification_CompleteCheck_OneRejected(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	checks := v.Checks()
	require.Len(t, checks, 3)

	// Approve first check
	completionTime := now.Add(time.Minute)
	v, err = v.CompleteCheck(checks[0].ID(), valueobject.StatusApproved, "", completionTime)
	require.NoError(t, err)

	// Reject second check
	v, err = v.CompleteCheck(checks[1].ID(), valueobject.StatusRejected, "document mismatch", completionTime)
	require.NoError(t, err)

	// Overall status should be REJECTED after a rejection
	assert.True(t, v.Status().Equal(valueobject.StatusRejected))

	// Should have emitted VerificationRejected event
	events := v.DomainEvents()
	found := false
	for _, e := range events {
		if e.EventType() == "identity.verification.rejected" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have emitted VerificationRejected event")
}

func TestIdentityVerification_CompleteCheck_Partial(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	checks := v.Checks()

	// Complete only the first check
	completionTime := now.Add(time.Minute)
	v, err = v.CompleteCheck(checks[0].ID(), valueobject.StatusApproved, "", completionTime)
	require.NoError(t, err)

	// Overall status should still be IN_PROGRESS (not all checks done)
	assert.True(t, v.Status().Equal(valueobject.StatusInProgress))
}

func TestIdentityVerification_CompleteCheck_AlreadyTerminal_Error(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	checks := v.Checks()
	completionTime := now.Add(time.Minute)

	// Complete all checks as APPROVED
	for _, c := range checks {
		v, err = v.CompleteCheck(c.ID(), valueobject.StatusApproved, "", completionTime)
		require.NoError(t, err)
	}
	assert.True(t, v.Status().Equal(valueobject.StatusApproved))

	// Try to complete another check on a terminal verification
	_, err = v.CompleteCheck(checks[0].ID(), valueobject.StatusRejected, "too late", completionTime)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in terminal status")
}

func TestIdentityVerification_CompleteCheck_NotFound_Error(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	// Try to complete a non-existent check
	_, err = v.CompleteCheck(uuid.New(), valueobject.StatusApproved, "", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIdentityVerification_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	createdAt := time.Date(2024, time.March, 14, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.March, 14, 11, 0, 0, 0, time.UTC)

	check := model.ReconstructCheck(
		uuid.New(),
		valueobject.CheckTypeDocument,
		valueobject.StatusApproved,
		"persona",
		"ref-123",
		&updatedAt,
		"",
	)

	v := model.Reconstruct(
		id, tenantID,
		"Jane", "Smith", "jane@example.com", "1985-06-20", "GB",
		valueobject.StatusApproved,
		[]model.VerificationCheck{check},
		3, createdAt, updatedAt,
	)

	assert.Equal(t, id, v.ID())
	assert.Equal(t, tenantID, v.TenantID())
	assert.Equal(t, "Jane", v.ApplicantFirstName())
	assert.Equal(t, "Smith", v.ApplicantLastName())
	assert.Equal(t, "jane@example.com", v.ApplicantEmail())
	assert.Equal(t, "1985-06-20", v.ApplicantDOB())
	assert.Equal(t, "GB", v.ApplicantCountry())
	assert.True(t, v.Status().Equal(valueobject.StatusApproved))
	assert.Len(t, v.Checks(), 1)
	assert.Equal(t, 3, v.Version())
	assert.Equal(t, createdAt, v.CreatedAt())
	assert.Equal(t, updatedAt, v.UpdatedAt())
	assert.Empty(t, v.DomainEvents())
}

func TestIdentityVerification_Immutability_StartProcessingDoesNotMutateOriginal(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	originalVersion := v.Version()
	originalStatus := v.Status()

	now := time.Now().UTC()
	_, err = v.StartProcessing(now)
	require.NoError(t, err)

	// Original must not have changed
	assert.Equal(t, originalVersion, v.Version())
	assert.True(t, originalStatus.Equal(v.Status()))
}

func TestIdentityVerification_Immutability_CompleteCheckDoesNotMutateOriginal(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	now := time.Now().UTC()
	processing, err := v.StartProcessing(now)
	require.NoError(t, err)

	originalVersion := processing.Version()
	checks := processing.Checks()

	_, err = processing.CompleteCheck(checks[0].ID(), valueobject.StatusApproved, "", now)
	require.NoError(t, err)

	// processing must not have changed
	assert.Equal(t, originalVersion, processing.Version())
}

func TestIdentityVerification_UpdateCheckProvider(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	checks := v.Checks()
	updated, err := v.UpdateCheckProvider(checks[0].ID(), "persona", "ref-abc123")
	require.NoError(t, err)

	updatedChecks := updated.Checks()
	assert.Equal(t, "persona", updatedChecks[0].Provider())
	assert.Equal(t, "ref-abc123", updatedChecks[0].ProviderReference())

	// Original unchanged
	assert.Empty(t, v.Checks()[0].Provider())
}

func TestIdentityVerification_UpdateCheckProvider_NotFound(t *testing.T) {
	tenantID := uuid.New()
	v, err := model.NewIdentityVerification(tenantID, "John", "Doe", "john@example.com", "1990-01-15", "US")
	require.NoError(t, err)

	_, err = v.UpdateCheckProvider(uuid.New(), "persona", "ref-abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIdentityVerification_FullLifecycle_Approved(t *testing.T) {
	tenantID := uuid.New()

	// Step 1: Create verification
	v, err := model.NewIdentityVerification(tenantID, "Alice", "Wonderland", "alice@example.com", "1995-03-25", "US")
	require.NoError(t, err)
	assert.True(t, v.Status().Equal(valueobject.StatusPending))

	// Step 2: Start processing
	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)
	assert.True(t, v.Status().Equal(valueobject.StatusInProgress))

	// Step 3: Complete all checks as approved
	checks := v.Checks()
	completionTime := now.Add(5 * time.Minute)
	for _, c := range checks {
		v, err = v.CompleteCheck(c.ID(), valueobject.StatusApproved, "", completionTime)
		require.NoError(t, err)
	}

	// Step 4: Verify final state
	assert.True(t, v.Status().Equal(valueobject.StatusApproved))
	assert.Equal(t, 5, v.Version()) // 1 (create) + 1 (start) + 3 (complete checks)

	// Verify domain events: initiated + completed
	events := v.DomainEvents()
	eventTypes := make(map[string]bool)
	for _, e := range events {
		eventTypes[e.EventType()] = true
	}
	assert.True(t, eventTypes["identity.verification.initiated"])
	assert.True(t, eventTypes["identity.verification.completed"])
}

func TestIdentityVerification_FullLifecycle_Rejected(t *testing.T) {
	tenantID := uuid.New()

	// Step 1: Create verification
	v, err := model.NewIdentityVerification(tenantID, "Bob", "Builder", "bob@example.com", "1988-12-01", "GB")
	require.NoError(t, err)

	// Step 2: Start processing
	now := time.Now().UTC()
	v, err = v.StartProcessing(now)
	require.NoError(t, err)

	checks := v.Checks()
	completionTime := now.Add(3 * time.Minute)

	// Step 3: Approve first check, reject second
	v, err = v.CompleteCheck(checks[0].ID(), valueobject.StatusApproved, "", completionTime)
	require.NoError(t, err)

	v, err = v.CompleteCheck(checks[1].ID(), valueobject.StatusRejected, "selfie does not match document", completionTime)
	require.NoError(t, err)

	// Step 4: Verify rejection
	assert.True(t, v.Status().Equal(valueobject.StatusRejected))

	// Verify domain events: initiated + rejected
	events := v.DomainEvents()
	eventTypes := make(map[string]bool)
	for _, e := range events {
		eventTypes[e.EventType()] = true
	}
	assert.True(t, eventTypes["identity.verification.initiated"])
	assert.True(t, eventTypes["identity.verification.rejected"])
	assert.False(t, eventTypes["identity.verification.completed"])
}

func TestVerificationCheck_Complete(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeDocument)
	assert.True(t, check.Status().Equal(valueobject.StatusPending))

	now := time.Now().UTC()
	completed, err := check.Complete(valueobject.StatusApproved, "", now)
	require.NoError(t, err)

	assert.True(t, completed.Status().Equal(valueobject.StatusApproved))
	assert.NotNil(t, completed.CompletedAt())
	assert.Empty(t, completed.FailureReason())

	// Original unchanged
	assert.True(t, check.Status().Equal(valueobject.StatusPending))
	assert.Nil(t, check.CompletedAt())
}

func TestVerificationCheck_Complete_WithFailure(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeSelfie)

	now := time.Now().UTC()
	completed, err := check.Complete(valueobject.StatusRejected, "face mismatch", now)
	require.NoError(t, err)

	assert.True(t, completed.Status().Equal(valueobject.StatusRejected))
	assert.Equal(t, "face mismatch", completed.FailureReason())
}

func TestVerificationCheck_Complete_AlreadyTerminal_Error(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeDocument)

	now := time.Now().UTC()
	completed, err := check.Complete(valueobject.StatusApproved, "", now)
	require.NoError(t, err)

	_, err = completed.Complete(valueobject.StatusRejected, "should fail", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in terminal status")
}

func TestVerificationCheck_Complete_NonTerminalStatus_Error(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeDocument)

	now := time.Now().UTC()
	_, err := check.Complete(valueobject.StatusInProgress, "", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "completion status must be terminal")
}

func TestVerificationCheck_SetInProgress(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeWatchlist)
	assert.True(t, check.Status().Equal(valueobject.StatusPending))

	started, err := check.SetInProgress()
	require.NoError(t, err)
	assert.True(t, started.Status().Equal(valueobject.StatusInProgress))

	// Original unchanged
	assert.True(t, check.Status().Equal(valueobject.StatusPending))
}

func TestVerificationCheck_SetInProgress_FromNonPending_Error(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeWatchlist)
	started, err := check.SetInProgress()
	require.NoError(t, err)

	_, err = started.SetInProgress()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only start processing checks in PENDING status")
}

func TestVerificationCheck_SetProvider(t *testing.T) {
	check := model.NewVerificationCheck(valueobject.CheckTypeDocument)

	updated := check.SetProvider("persona", "ref-123")
	assert.Equal(t, "persona", updated.Provider())
	assert.Equal(t, "ref-123", updated.ProviderReference())

	// Original unchanged
	assert.Empty(t, check.Provider())
	assert.Empty(t, check.ProviderReference())
}
