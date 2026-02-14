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

func newTestLoan(t *testing.T) model.Loan {
	t.Helper()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	loan, err := model.NewLoan(
		"tenant-1", "app-1", "account-1",
		decimal.NewFromInt(100_000), "USD",
		500, 360, now,
	)
	require.NoError(t, err)
	return loan
}

func TestLoan_Creation(t *testing.T) {
	loan := newTestLoan(t)

	assert.NotEmpty(t, loan.ID())
	assert.Equal(t, "tenant-1", loan.TenantID())
	assert.Equal(t, "app-1", loan.ApplicationID())
	assert.Equal(t, "account-1", loan.BorrowerAccountID())
	assert.True(t, loan.Principal().Equal(decimal.NewFromInt(100_000)))
	assert.Equal(t, "USD", loan.Currency())
	assert.Equal(t, 500, loan.InterestRateBps())
	assert.Equal(t, 360, loan.TermMonths())
	assert.True(t, loan.Status().Equal(valueobject.LoanStatusActive))
	assert.True(t, loan.OutstandingBalance().Equal(decimal.NewFromInt(100_000)))
	assert.Len(t, loan.Schedule(), 360)
	assert.Equal(t, 1, loan.Version())
	assert.Len(t, loan.DomainEvents(), 1, "should have LoanDisbursed event")
}

func TestLoan_MakePayment(t *testing.T) {
	loan := newTestLoan(t)

	// Make a $1000 payment.
	updated, err := loan.MakePayment(decimal.NewFromInt(1_000), time.Now().UTC())
	require.NoError(t, err)

	expectedBalance := decimal.NewFromInt(99_000)
	assert.True(t, updated.OutstandingBalance().Equal(expectedBalance),
		"outstanding should be $99,000, got %s", updated.OutstandingBalance())
	assert.True(t, updated.Status().Equal(valueobject.LoanStatusActive))
	assert.Len(t, updated.DomainEvents(), 2, "should have disbursed + payment_received")
}

func TestLoan_MakePayment_FullPayoff(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	loan, err := model.NewLoan("t-1", "app-1", "acc-1",
		decimal.NewFromInt(5_000), "USD", 500, 12, now)
	require.NoError(t, err)

	// Pay off the entire loan.
	updated, err := loan.MakePayment(decimal.NewFromInt(5_000), now)
	require.NoError(t, err)
	assert.True(t, updated.OutstandingBalance().Equal(decimal.Zero))
	assert.True(t, updated.Status().Equal(valueobject.LoanStatusPaidOff))
}

func TestLoan_MakePayment_Errors(t *testing.T) {
	loan := newTestLoan(t)

	t.Run("zero amount", func(t *testing.T) {
		_, err := loan.MakePayment(decimal.Zero, time.Now())
		assert.Error(t, err)
	})

	t.Run("negative amount", func(t *testing.T) {
		_, err := loan.MakePayment(decimal.NewFromInt(-100), time.Now())
		assert.Error(t, err)
	})

	t.Run("exceeds balance", func(t *testing.T) {
		_, err := loan.MakePayment(decimal.NewFromInt(200_000), time.Now())
		assert.Error(t, err)
	})
}

func TestLoan_Delinquency(t *testing.T) {
	loan := newTestLoan(t)
	now := time.Now().UTC()

	// Mark delinquent.
	delinquent, err := loan.MarkDelinquent(now)
	require.NoError(t, err)
	assert.True(t, delinquent.Status().Equal(valueobject.LoanStatusDelinquent))

	// Payment allowed on delinquent loan.
	paid, err := delinquent.MakePayment(decimal.NewFromInt(500), now)
	require.NoError(t, err)
	assert.True(t, paid.OutstandingBalance().Equal(decimal.NewFromInt(99_500)))

	// Mark default from delinquent.
	defaulted, err := delinquent.MarkDefault(now)
	require.NoError(t, err)
	assert.True(t, defaulted.Status().Equal(valueobject.LoanStatusDefault))

	// Write off from default.
	written, err := defaulted.WriteOff(now)
	require.NoError(t, err)
	assert.True(t, written.Status().Equal(valueobject.LoanStatusWrittenOff))
}

func TestLoan_InvalidTransitions(t *testing.T) {
	loan := newTestLoan(t)
	now := time.Now().UTC()

	// Cannot mark default directly from active.
	_, err := loan.MarkDefault(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// Cannot write off from active.
	_, err = loan.WriteOff(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// Cannot mark delinquent twice.
	delinquent, err := loan.MarkDelinquent(now)
	require.NoError(t, err)
	_, err = delinquent.MarkDelinquent(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)

	// Cannot write off from delinquent (must go through default first).
	_, err = delinquent.WriteOff(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)
}

func TestLoan_PayOff(t *testing.T) {
	loan := newTestLoan(t)
	now := time.Now().UTC()

	paidOff, err := loan.PayOff(now)
	require.NoError(t, err)
	assert.True(t, paidOff.Status().Equal(valueobject.LoanStatusPaidOff))
	assert.True(t, paidOff.OutstandingBalance().Equal(decimal.Zero))

	// Cannot pay off again.
	_, err = paidOff.PayOff(now)
	assert.ErrorIs(t, err, valueobject.ErrInvalidStatusTransition)
}

func TestLoan_ValidationErrors(t *testing.T) {
	now := time.Now().UTC()

	t.Run("empty tenant", func(t *testing.T) {
		_, err := model.NewLoan("", "app", "acc", decimal.NewFromInt(1000), "USD", 500, 12, now)
		assert.Error(t, err)
	})

	t.Run("empty application", func(t *testing.T) {
		_, err := model.NewLoan("t", "", "acc", decimal.NewFromInt(1000), "USD", 500, 12, now)
		assert.Error(t, err)
	})

	t.Run("empty borrower", func(t *testing.T) {
		_, err := model.NewLoan("t", "app", "", decimal.NewFromInt(1000), "USD", 500, 12, now)
		assert.Error(t, err)
	})

	t.Run("zero principal", func(t *testing.T) {
		_, err := model.NewLoan("t", "app", "acc", decimal.Zero, "USD", 500, 12, now)
		assert.Error(t, err)
	})

	t.Run("empty currency", func(t *testing.T) {
		_, err := model.NewLoan("t", "app", "acc", decimal.NewFromInt(1000), "", 500, 12, now)
		assert.Error(t, err)
	})

	t.Run("zero term", func(t *testing.T) {
		_, err := model.NewLoan("t", "app", "acc", decimal.NewFromInt(1000), "USD", 500, 0, now)
		assert.Error(t, err)
	})
}

func TestLoan_ScheduleDefensiveCopy(t *testing.T) {
	loan := newTestLoan(t)

	s1 := loan.Schedule()
	s2 := loan.Schedule()

	require.NotEmpty(t, s1)
	require.NotEmpty(t, s2)

	// Mutating the returned slice should not affect the loan.
	s1[0].Period = 9999
	assert.NotEqual(t, 9999, s2[0].Period, "schedule should be a defensive copy")
}
