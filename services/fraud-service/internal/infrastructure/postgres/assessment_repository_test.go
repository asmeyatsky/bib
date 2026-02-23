package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAssessmentRepository tests the constructor.
func TestNewAssessmentRepository(t *testing.T) {
	t.Run("creates repository with nil pool", func(t *testing.T) {
		repo := NewAssessmentRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}
