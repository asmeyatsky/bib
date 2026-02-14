package tests

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

func TestLoanApplication_FullLifecycle_Approved(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	// 1. Create a new application.
	app, err := model.NewLoanApplication(
		"tenant-1", "applicant-1",
		decimal.NewFromInt(50_000), "USD", 60, "home renovation", now,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, app.ID())
	assert.Equal(t, "tenant-1", app.TenantID())
	assert.Equal(t, "applicant-1", app.ApplicantID())
	assert.True(t, app.RequestedAmount().Equal(decimal.NewFromInt(50_000)))
	assert.Equal(t, "USD", app.Currency())
	assert.Equal(t, 60, app.TermMonths())
	assert.Equal(t, "home renovation", app.Purpose())
	assert.True(t, app.Status().Equal(valueobject.LoanApplicationStatusSubmitted))
	assert.Equal(t, 1, app.Version())
	assert.Len(t, app.DomainEvents(), 1, "should have LoanApplicationSubmitted event")

	// 2. Submit for review.
	app, err = app.SubmitForReview(now)
	require.NoError(t, err)
	assert.True(t, app.Status().Equal(valueobject.LoanApplicationStatusUnderReview))

	// 3. Approve.
	app, err = app.Approve("good credit tier", "720", now)
	require.NoError(t, err)
	assert.True(t, app.Status().Equal(valueobject.LoanApplicationStatusApproved))
	assert.Equal(t, "good credit tier", app.DecisionReason())
	assert.Equal(t, "720", app.CreditScore())
	assert.Len(t, app.DomainEvents(), 3, "should have submitted + approved events")

	// 4. Mark disbursed.
	app, err = app.MarkDisbursed(now)
	require.NoError(t, err)
	assert.True(t, app.Status().Equal(valueobject.LoanApplicationStatusDisbursed))

	// 5. Clear events.
	app = app.ClearEvents()
	assert.Empty(t, app.DomainEvents())
}

func TestLoanApplication_FullLifecycle_Rejected(t *testing.T) {
	now := time.Now().UTC()

	app, err := model.NewLoanApplication(
		"tenant-1", "applicant-2",
		decimal.NewFromInt(200_000), "USD", 360, "purchase", now,
	)
	require.NoError(t, err)

	app, err = app.SubmitForReview(now)
	require.NoError(t, err)

	app, err = app.Reject("credit score too low", now)
	require.NoError(t, err)
	assert.True(t, app.Status().Equal(valueobject.LoanApplicationStatusRejected))
	assert.Equal(t, "credit score too low", app.DecisionReason())
}

func TestLoanApplication_InvalidTransitions(t *testing.T) {
	now := time.Now().UTC()

	app, err := model.NewLoanApplication(
		"tenant-1", "applicant-1",
		decimal.NewFromInt(10_000), "USD", 12, "test", now,
	)
	require.NoError(t, err)

	// Cannot approve directly from SUBMITTED.
	_, err = app.Approve("reason", "750", now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// Cannot reject directly from SUBMITTED.
	_, err = app.Reject("reason", now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// Cannot disburse from SUBMITTED.
	_, err = app.MarkDisbursed(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// After review, cannot submit again.
	reviewed, err := app.SubmitForReview(now)
	require.NoError(t, err)
	_, err = reviewed.SubmitForReview(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)
}

func TestLoanApplication_ValidationErrors(t *testing.T) {
	now := time.Now().UTC()

	t.Run("empty tenant", func(t *testing.T) {
		_, err := model.NewLoanApplication("", "app-1", decimal.NewFromInt(1000), "USD", 12, "test", now)
		assert.Error(t, err)
	})

	t.Run("empty applicant", func(t *testing.T) {
		_, err := model.NewLoanApplication("t-1", "", decimal.NewFromInt(1000), "USD", 12, "test", now)
		assert.Error(t, err)
	})

	t.Run("zero amount", func(t *testing.T) {
		_, err := model.NewLoanApplication("t-1", "a-1", decimal.Zero, "USD", 12, "test", now)
		assert.Error(t, err)
	})

	t.Run("negative amount", func(t *testing.T) {
		_, err := model.NewLoanApplication("t-1", "a-1", decimal.NewFromInt(-1000), "USD", 12, "test", now)
		assert.Error(t, err)
	})

	t.Run("empty currency", func(t *testing.T) {
		_, err := model.NewLoanApplication("t-1", "a-1", decimal.NewFromInt(1000), "", 12, "test", now)
		assert.Error(t, err)
	})

	t.Run("zero term", func(t *testing.T) {
		_, err := model.NewLoanApplication("t-1", "a-1", decimal.NewFromInt(1000), "USD", 0, "test", now)
		assert.Error(t, err)
	})
}
