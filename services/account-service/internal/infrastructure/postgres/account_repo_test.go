package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// TestReconstructAccount tests the reconstructAccount helper that maps
// raw database values back into the CustomerAccount aggregate.
func TestReconstructAccount(t *testing.T) {
	t.Run("successfully reconstructs account with identity verification", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		holderID := uuid.New()
		verificationID := uuid.New()
		now := time.Now().UTC().Truncate(time.Microsecond)

		account, err := reconstructAccount(
			id, tenantID,
			"BIB-ABCD-1234-WXYZ", "CHECKING", "ACTIVE",
			"USD", "2000-100",
			2, now, now,
			holderID, "Jane", "Smith", "jane@example.com", &verificationID,
		)

		require.NoError(t, err)
		assert.Equal(t, id, account.ID())
		assert.Equal(t, tenantID, account.TenantID())
		assert.Equal(t, "BIB-ABCD-1234-WXYZ", account.AccountNumber().String())
		assert.Equal(t, "CHECKING", account.AccountType().String())
		assert.Equal(t, model.AccountStatusActive, account.Status())
		assert.Equal(t, "USD", account.Currency())
		assert.Equal(t, "2000-100", account.LedgerAccountCode())
		assert.Equal(t, 2, account.Version())
		assert.Equal(t, now, account.CreatedAt())
		assert.Equal(t, now, account.UpdatedAt())
		assert.Equal(t, holderID, account.Holder().ID())
		assert.Equal(t, "Jane", account.Holder().FirstName())
		assert.Equal(t, "Smith", account.Holder().LastName())
		assert.Equal(t, "jane@example.com", account.Holder().Email())
		assert.Equal(t, verificationID, account.Holder().IdentityVerificationID())
	})

	t.Run("successfully reconstructs account without identity verification", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		holderID := uuid.New()
		now := time.Now().UTC().Truncate(time.Microsecond)

		account, err := reconstructAccount(
			id, tenantID,
			"BIB-ABCD-1234-WXYZ", "SAVINGS", "PENDING",
			"EUR", "2100-200",
			1, now, now,
			holderID, "John", "Doe", "john@example.com", nil,
		)

		require.NoError(t, err)
		assert.Equal(t, model.AccountStatusPending, account.Status())
		assert.Equal(t, "SAVINGS", account.AccountType().String())
		assert.Equal(t, uuid.Nil, account.Holder().IdentityVerificationID())
	})

	t.Run("returns error for invalid account number format", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		holderID := uuid.New()
		now := time.Now().UTC()

		_, err := reconstructAccount(
			id, tenantID,
			"INVALID-NUMBER", "CHECKING", "ACTIVE",
			"USD", "2000-100",
			1, now, now,
			holderID, "Jane", "Smith", "jane@example.com", nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid stored account number")
	})

	t.Run("returns error for invalid account type", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		holderID := uuid.New()
		now := time.Now().UTC()

		_, err := reconstructAccount(
			id, tenantID,
			"BIB-ABCD-1234-WXYZ", "INVALID_TYPE", "ACTIVE",
			"USD", "2000-100",
			1, now, now,
			holderID, "Jane", "Smith", "jane@example.com", nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid stored account type")
	})

	t.Run("reconstructs all supported account statuses", func(t *testing.T) {
		statuses := []struct {
			input    string
			expected model.AccountStatus
		}{
			{"PENDING", model.AccountStatusPending},
			{"ACTIVE", model.AccountStatusActive},
			{"FROZEN", model.AccountStatusFrozen},
			{"CLOSED", model.AccountStatusClosed},
		}

		for _, tc := range statuses {
			t.Run(tc.input, func(t *testing.T) {
				id := uuid.New()
				tenantID := uuid.New()
				holderID := uuid.New()
				now := time.Now().UTC()

				account, err := reconstructAccount(
					id, tenantID,
					"BIB-ABCD-1234-WXYZ", "CHECKING", tc.input,
					"USD", "2000-100",
					1, now, now,
					holderID, "Jane", "Smith", "jane@example.com", nil,
				)

				require.NoError(t, err)
				assert.Equal(t, tc.expected, account.Status())
			})
		}
	})

	t.Run("reconstructs all supported account types", func(t *testing.T) {
		accountTypes := []string{"CHECKING", "SAVINGS", "LOAN", "NOMINAL"}

		for _, at := range accountTypes {
			t.Run(at, func(t *testing.T) {
				id := uuid.New()
				tenantID := uuid.New()
				holderID := uuid.New()
				now := time.Now().UTC()

				account, err := reconstructAccount(
					id, tenantID,
					"BIB-ABCD-1234-WXYZ", at, "ACTIVE",
					"USD", "2000-100",
					1, now, now,
					holderID, "Jane", "Smith", "jane@example.com", nil,
				)

				require.NoError(t, err)
				assert.Equal(t, at, account.AccountType().String())
			})
		}
	})
}

// TestNewAccountRepository tests the constructor.
func TestNewAccountRepository(t *testing.T) {
	t.Run("creates repository with nil pool", func(t *testing.T) {
		repo := NewAccountRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}

// TestAccountNumberRoundTrip tests that an account number survives serialization.
func TestAccountNumberRoundTrip(t *testing.T) {
	original := valueobject.NewAccountNumber()
	reconstructed, err := valueobject.AccountNumberFromString(original.String())
	require.NoError(t, err)
	assert.True(t, original.Equal(reconstructed))
}
