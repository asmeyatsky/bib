package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

func TestNewPaymentStatus_ValidStatuses(t *testing.T) {
	tests := []struct {
		input    string
		expected valueobject.PaymentStatus
	}{
		{"INITIATED", valueobject.PaymentStatusInitiated},
		{"PROCESSING", valueobject.PaymentStatusProcessing},
		{"SETTLED", valueobject.PaymentStatusSettled},
		{"FAILED", valueobject.PaymentStatusFailed},
		{"REVERSED", valueobject.PaymentStatusReversed},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			status, err := valueobject.NewPaymentStatus(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, status)
			assert.Equal(t, tc.input, status.String())
			assert.False(t, status.IsZero())
		})
	}
}

func TestNewPaymentStatus_InvalidStatus(t *testing.T) {
	invalidStatuses := []string{"", "INVALID", "pending", "Settled", "CANCELED"}

	for _, input := range invalidStatuses {
		t.Run(input, func(t *testing.T) {
			_, err := valueobject.NewPaymentStatus(input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment status")
		})
	}
}

func TestPaymentStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   valueobject.PaymentStatus
		terminal bool
	}{
		{valueobject.PaymentStatusInitiated, false},
		{valueobject.PaymentStatusProcessing, false},
		{valueobject.PaymentStatusSettled, true},
		{valueobject.PaymentStatusFailed, true},
		{valueobject.PaymentStatusReversed, true},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			assert.Equal(t, tc.terminal, tc.status.IsTerminal())
		})
	}
}

func TestPaymentStatus_IsZero(t *testing.T) {
	var zeroStatus valueobject.PaymentStatus
	assert.True(t, zeroStatus.IsZero())
	assert.Equal(t, "", zeroStatus.String())
	assert.False(t, zeroStatus.IsTerminal())
}
