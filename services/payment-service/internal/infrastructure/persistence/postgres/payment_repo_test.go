package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewPaymentOrderRepo tests the constructor.
func TestNewPaymentOrderRepo(t *testing.T) {
	t.Run("creates repo with nil pool", func(t *testing.T) {
		repo := NewPaymentOrderRepo(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.pool)
	})
}

// TestPaymentOrderRepoImplementsInterface confirms the compile-time interface check.
func TestPaymentOrderRepoImplementsInterface(t *testing.T) {
	// The source file has:
	//   var _ port.PaymentOrderRepository = (*PaymentOrderRepo)(nil)
	repo := NewPaymentOrderRepo(nil)
	assert.NotNil(t, repo)
}
