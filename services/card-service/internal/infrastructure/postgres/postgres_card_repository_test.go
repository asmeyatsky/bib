package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewPostgresCardRepository tests the constructor.
func TestNewPostgresCardRepository(t *testing.T) {
	t.Run("creates repository with nil pool", func(t *testing.T) {
		repo := NewPostgresCardRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}
