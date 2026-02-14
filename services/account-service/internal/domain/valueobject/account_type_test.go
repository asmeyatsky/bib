package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

func TestNewAccountType(t *testing.T) {
	t.Run("accepts CHECKING", func(t *testing.T) {
		at, err := valueobject.NewAccountType("CHECKING")
		require.NoError(t, err)
		assert.Equal(t, "CHECKING", at.String())
		assert.True(t, at.Equal(valueobject.AccountTypeChecking))
	})

	t.Run("accepts SAVINGS", func(t *testing.T) {
		at, err := valueobject.NewAccountType("SAVINGS")
		require.NoError(t, err)
		assert.Equal(t, "SAVINGS", at.String())
		assert.True(t, at.Equal(valueobject.AccountTypeSavings))
	})

	t.Run("accepts LOAN", func(t *testing.T) {
		at, err := valueobject.NewAccountType("LOAN")
		require.NoError(t, err)
		assert.Equal(t, "LOAN", at.String())
		assert.True(t, at.Equal(valueobject.AccountTypeLoan))
	})

	t.Run("accepts NOMINAL", func(t *testing.T) {
		at, err := valueobject.NewAccountType("NOMINAL")
		require.NoError(t, err)
		assert.Equal(t, "NOMINAL", at.String())
		assert.True(t, at.Equal(valueobject.AccountTypeNominal))
	})

	t.Run("rejects unknown type", func(t *testing.T) {
		_, err := valueobject.NewAccountType("UNKNOWN")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown account type")
	})

	t.Run("rejects empty string", func(t *testing.T) {
		_, err := valueobject.NewAccountType("")
		assert.Error(t, err)
	})

	t.Run("rejects lowercase", func(t *testing.T) {
		_, err := valueobject.NewAccountType("checking")
		assert.Error(t, err)
	})
}

func TestAccountType_IsZero(t *testing.T) {
	t.Run("zero value is zero", func(t *testing.T) {
		var at valueobject.AccountType
		assert.True(t, at.IsZero())
	})

	t.Run("valid type is not zero", func(t *testing.T) {
		at, _ := valueobject.NewAccountType("CHECKING")
		assert.False(t, at.IsZero())
	})
}

func TestAccountType_Equal(t *testing.T) {
	t.Run("same types are equal", func(t *testing.T) {
		at1, _ := valueobject.NewAccountType("CHECKING")
		at2, _ := valueobject.NewAccountType("CHECKING")
		assert.True(t, at1.Equal(at2))
	})

	t.Run("different types are not equal", func(t *testing.T) {
		at1, _ := valueobject.NewAccountType("CHECKING")
		at2, _ := valueobject.NewAccountType("SAVINGS")
		assert.False(t, at1.Equal(at2))
	})

	t.Run("predefined constants match", func(t *testing.T) {
		assert.True(t, valueobject.AccountTypeChecking.Equal(valueobject.AccountTypeChecking))
		assert.False(t, valueobject.AccountTypeChecking.Equal(valueobject.AccountTypeSavings))
	})
}
