package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

func validXBRL() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance">
  <xbrli:context id="ctx_2025-Q1">
    <xbrli:entity>
      <xbrli:identifier scheme="http://www.bibbank.com">test</xbrli:identifier>
    </xbrli:entity>
    <xbrli:period>
      <xbrli:instant>2025-03-31</xbrli:instant>
    </xbrli:period>
  </xbrli:context>
</xbrli:xbrl>`
}

func TestNewReportSubmission(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates submission in DRAFT status", func(t *testing.T) {
		sub, err := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, sub.ID())
		assert.Equal(t, tenantID, sub.TenantID())
		assert.True(t, sub.ReportType().Equal(valueobject.ReportTypeCOREP))
		assert.Equal(t, "2025-Q1", sub.ReportingPeriod())
		assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusDraft))
		assert.Equal(t, "", sub.XBRLContent())
		assert.Nil(t, sub.GeneratedAt())
		assert.Nil(t, sub.SubmittedAt())
		assert.Empty(t, sub.ValidationErrors())
		assert.Equal(t, 1, sub.Version())
	})

	t.Run("rejects nil tenant ID", func(t *testing.T) {
		_, err := model.NewReportSubmission(uuid.Nil, valueobject.ReportTypeCOREP, "2025-Q1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant ID")
	})

	t.Run("rejects empty report type", func(t *testing.T) {
		_, err := model.NewReportSubmission(tenantID, valueobject.ReportType{}, "2025-Q1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "report type")
	})

	t.Run("rejects empty period", func(t *testing.T) {
		_, err := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "period")
	})
}

func TestReportSubmission_FullLifecycle_Accept(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	// Step 1: Create in DRAFT.
	sub, err := model.NewReportSubmission(tenantID, valueobject.ReportTypeFINREP, "2025-Q1")
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusDraft))

	// Step 2: Mark GENERATING.
	sub, err = sub.MarkGenerating(now)
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusGenerating))

	// Step 3: Set generated XBRL content.
	xbrl := validXBRL()
	genTime := now.Add(5 * time.Second)
	sub, err = sub.SetGenerated(xbrl, genTime)
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusReady))
	assert.Equal(t, xbrl, sub.XBRLContent())
	assert.NotNil(t, sub.GeneratedAt())

	// Verify ReportGenerated event was emitted.
	events := sub.DomainEvents()
	require.Len(t, events, 1)
	genEvent, ok := events[0].(event.ReportGenerated)
	require.True(t, ok)
	assert.Equal(t, sub.ID(), genEvent.ID)
	assert.Equal(t, "report.generated", genEvent.EventType())

	// Step 4: Validate.
	sub, err = sub.Validate()
	require.NoError(t, err)
	assert.Empty(t, sub.ValidationErrors())

	// Step 5: Submit.
	submitTime := now.Add(10 * time.Second)
	sub, err = sub.Submit(submitTime)
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusSubmitted))
	assert.NotNil(t, sub.SubmittedAt())

	// Verify ReportSubmitted event was emitted.
	events = sub.DomainEvents()
	require.Len(t, events, 2)
	subEvent, ok := events[1].(event.ReportSubmitted)
	require.True(t, ok)
	assert.Equal(t, sub.ID(), subEvent.ID)

	// Step 6: Accept.
	acceptTime := now.Add(60 * time.Second)
	sub, err = sub.Accept(acceptTime)
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusAccepted))

	// Verify ReportAccepted event.
	events = sub.DomainEvents()
	require.Len(t, events, 3)
	accEvent, ok := events[2].(event.ReportAccepted)
	require.True(t, ok)
	assert.Equal(t, sub.ID(), accEvent.ID)
}

func TestReportSubmission_FullLifecycle_Reject(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	// Create -> Generate -> SetGenerated -> Validate -> Submit -> Reject.
	sub, err := model.NewReportSubmission(tenantID, valueobject.ReportTypeMREL, "2025-Q2")
	require.NoError(t, err)

	sub, err = sub.MarkGenerating(now)
	require.NoError(t, err)

	sub, err = sub.SetGenerated(validXBRL(), now.Add(5*time.Second))
	require.NoError(t, err)

	sub, err = sub.Validate()
	require.NoError(t, err)

	sub, err = sub.Submit(now.Add(10 * time.Second))
	require.NoError(t, err)

	// Reject with errors.
	rejErrors := []string{"CET1 ratio below minimum threshold", "Missing MREL disclosure"}
	sub, err = sub.Reject(rejErrors, now.Add(60*time.Second))
	require.NoError(t, err)
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusRejected))
	assert.Equal(t, rejErrors, sub.ValidationErrors())

	// Verify ReportRejected event.
	events := sub.DomainEvents()
	require.Len(t, events, 3) // Generated + Submitted + Rejected
	rejEvent, ok := events[2].(event.ReportRejected)
	require.True(t, ok)
	assert.Equal(t, rejErrors, rejEvent.ValidationErrors)
}

func TestReportSubmission_InvalidTransitions(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	t.Run("cannot mark generating from non-DRAFT", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		sub, _ = sub.MarkGenerating(now)
		_, err := sub.MarkGenerating(now) // already GENERATING
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DRAFT")
	})

	t.Run("cannot set generated from non-GENERATING", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		_, err := sub.SetGenerated(validXBRL(), now) // still DRAFT
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GENERATING")
	})

	t.Run("cannot set generated with empty content", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		sub, _ = sub.MarkGenerating(now)
		_, err := sub.SetGenerated("", now)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("cannot submit from non-READY", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		_, err := sub.Submit(now) // still DRAFT
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "READY")
	})

	t.Run("cannot accept from non-SUBMITTED", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		_, err := sub.Accept(now) // still DRAFT
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBMITTED")
	})

	t.Run("cannot reject from non-SUBMITTED", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		_, err := sub.Reject([]string{"error"}, now) // still DRAFT
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBMITTED")
	})

	t.Run("cannot reject without errors", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		sub, _ = sub.MarkGenerating(now)
		sub, _ = sub.SetGenerated(validXBRL(), now)
		sub, _ = sub.Submit(now)
		_, err := sub.Reject([]string{}, now)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one error")
	})
}

func TestReportSubmission_Validate(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	t.Run("valid XBRL passes validation", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		sub, _ = sub.MarkGenerating(now)
		sub, _ = sub.SetGenerated(validXBRL(), now)

		sub, err := sub.Validate()
		require.NoError(t, err)
		assert.Empty(t, sub.ValidationErrors())
	})

	t.Run("invalid XBRL fails validation", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		sub, _ = sub.MarkGenerating(now)
		sub, _ = sub.SetGenerated("<not-valid-xbrl/>", now)

		_, err := sub.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("cannot validate from non-READY status", func(t *testing.T) {
		sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
		_, err := sub.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "READY")
	})
}

func TestReportSubmission_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()
	genAt := now.Add(-5 * time.Minute)
	subAt := now.Add(-1 * time.Minute)

	sub := model.Reconstruct(
		id, tenantID, valueobject.ReportTypeFINREP, "2025-Q3",
		valueobject.SubmissionStatusSubmitted, "<xbrl/>",
		&genAt, &subAt, []string{}, 3, now.Add(-10*time.Minute), now,
	)

	assert.Equal(t, id, sub.ID())
	assert.Equal(t, tenantID, sub.TenantID())
	assert.True(t, sub.ReportType().Equal(valueobject.ReportTypeFINREP))
	assert.Equal(t, "2025-Q3", sub.ReportingPeriod())
	assert.True(t, sub.Status().Equal(valueobject.SubmissionStatusSubmitted))
	assert.Equal(t, "<xbrl/>", sub.XBRLContent())
	assert.NotNil(t, sub.GeneratedAt())
	assert.NotNil(t, sub.SubmittedAt())
	assert.Equal(t, 3, sub.Version())
	assert.Empty(t, sub.DomainEvents())
}

func TestReportSubmission_ClearDomainEvents(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	sub, _ := model.NewReportSubmission(tenantID, valueobject.ReportTypeCOREP, "2025-Q1")
	sub, _ = sub.MarkGenerating(now)
	sub, _ = sub.SetGenerated(validXBRL(), now)

	require.Len(t, sub.DomainEvents(), 1)

	sub = sub.ClearDomainEvents()
	assert.Empty(t, sub.DomainEvents())
}
