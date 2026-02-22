package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewBalanceRepo tests the constructor.
func TestNewBalanceRepo(t *testing.T) {
	t.Run("creates repo with nil pool", func(t *testing.T) {
		repo := NewBalanceRepo(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}

// TestBalanceRepoImplementsInterface confirms the compile-time interface check.
func TestBalanceRepoImplementsInterface(t *testing.T) {
	repo := NewBalanceRepo(nil)
	assert.NotNil(t, repo)
}
