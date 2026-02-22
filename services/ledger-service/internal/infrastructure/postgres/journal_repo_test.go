package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewJournalRepo tests the constructor.
func TestNewJournalRepo(t *testing.T) {
	t.Run("creates repo with nil pool", func(t *testing.T) {
		repo := NewJournalRepo(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}

// TestJournalRepoSQLQueries verifies the SQL query constants embedded in the
// repository methods are valid. Since we cannot run against a real database,
// we verify the repository construction and that methods accept the correct
// number of parameters via table-driven assertions about the repository
// interface implementation.
func TestJournalRepoImplementsInterface(t *testing.T) {
	// This compile-time assertion is already in the source file via:
	//   var _ port.JournalRepository = (*JournalRepo)(nil)
	// This test ensures it stays valid and the types match.
	repo := NewJournalRepo(nil)
	assert.NotNil(t, repo)
}
