package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

func TestNewAccountNumber(t *testing.T) {
	t.Run("generates a valid account number", func(t *testing.T) {
		num := valueobject.NewAccountNumber()
		assert.False(t, num.IsZero())
		assert.Regexp(t, `^BIB-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`, num.String())
	})

	t.Run("generates unique account numbers", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			num := valueobject.NewAccountNumber()
			assert.False(t, seen[num.String()], "duplicate account number generated: %s", num.String())
			seen[num.String()] = true
		}
	})
}

func TestAccountNumberFromString(t *testing.T) {
	t.Run("accepts valid format", func(t *testing.T) {
		num, err := valueobject.AccountNumberFromString("BIB-A1B2-C3D4-E5F6")
		require.NoError(t, err)
		assert.Equal(t, "BIB-A1B2-C3D4-E5F6", num.String())
	})

	t.Run("trims whitespace", func(t *testing.T) {
		num, err := valueobject.AccountNumberFromString("  BIB-AAAA-BBBB-CCCC  ")
		require.NoError(t, err)
		assert.Equal(t, "BIB-AAAA-BBBB-CCCC", num.String())
	})

	t.Run("rejects empty string", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("")
		assert.Error(t, err)
	})

	t.Run("rejects invalid prefix", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("XXX-A1B2-C3D4-E5F6")
		assert.Error(t, err)
	})

	t.Run("rejects wrong segment count", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("BIB-A1B2-C3D4")
		assert.Error(t, err)
	})

	t.Run("rejects lowercase letters", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("BIB-a1b2-c3d4-e5f6")
		assert.Error(t, err)
	})

	t.Run("rejects special characters", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("BIB-A1B!-C3D4-E5F6")
		assert.Error(t, err)
	})

	t.Run("rejects wrong segment length", func(t *testing.T) {
		_, err := valueobject.AccountNumberFromString("BIB-A1B-C3D4-E5F6")
		assert.Error(t, err)
	})
}

func TestAccountNumber_IsZero(t *testing.T) {
	t.Run("zero value is zero", func(t *testing.T) {
		var num valueobject.AccountNumber
		assert.True(t, num.IsZero())
	})

	t.Run("new account number is not zero", func(t *testing.T) {
		num := valueobject.NewAccountNumber()
		assert.False(t, num.IsZero())
	})
}

func TestAccountNumber_Equal(t *testing.T) {
	t.Run("equal account numbers", func(t *testing.T) {
		num1, _ := valueobject.AccountNumberFromString("BIB-AAAA-BBBB-CCCC")
		num2, _ := valueobject.AccountNumberFromString("BIB-AAAA-BBBB-CCCC")
		assert.True(t, num1.Equal(num2))
	})

	t.Run("different account numbers", func(t *testing.T) {
		num1, _ := valueobject.AccountNumberFromString("BIB-AAAA-BBBB-CCCC")
		num2, _ := valueobject.AccountNumberFromString("BIB-DDDD-EEEE-FFFF")
		assert.False(t, num1.Equal(num2))
	})
}
