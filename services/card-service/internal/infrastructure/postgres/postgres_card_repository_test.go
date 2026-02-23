package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewCardRepository tests the constructor.
func TestNewCardRepository(t *testing.T) {
	t.Run("creates repository with nil pool", func(t *testing.T) {
		repo := NewCardRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}
