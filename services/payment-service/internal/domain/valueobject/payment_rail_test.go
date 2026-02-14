package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

func TestNewPaymentRail_ValidRails(t *testing.T) {
	tests := []struct {
		input    string
		expected valueobject.PaymentRail
	}{
		{"ACH", valueobject.RailACH},
		{"FEDNOW", valueobject.RailFedNow},
		{"SWIFT", valueobject.RailSWIFT},
		{"SEPA", valueobject.RailSEPA},
		{"CHIPS", valueobject.RailCHIPS},
		{"INTERNAL", valueobject.RailInternal},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			rail, err := valueobject.NewPaymentRail(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, rail)
			assert.Equal(t, tc.input, rail.String())
			assert.False(t, rail.IsZero())
		})
	}
}

func TestNewPaymentRail_InvalidRail(t *testing.T) {
	invalidRails := []string{"", "INVALID", "ach", "Wire", "PAYPAL"}

	for _, input := range invalidRails {
		t.Run(input, func(t *testing.T) {
			_, err := valueobject.NewPaymentRail(input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment rail")
		})
	}
}

func TestPaymentRail_IsZero(t *testing.T) {
	var zeroRail valueobject.PaymentRail
	assert.True(t, zeroRail.IsZero())
	assert.Equal(t, "", zeroRail.String())
}

func TestPaymentRail_String(t *testing.T) {
	assert.Equal(t, "ACH", valueobject.RailACH.String())
	assert.Equal(t, "FEDNOW", valueobject.RailFedNow.String())
	assert.Equal(t, "SWIFT", valueobject.RailSWIFT.String())
	assert.Equal(t, "SEPA", valueobject.RailSEPA.String())
	assert.Equal(t, "CHIPS", valueobject.RailCHIPS.String())
	assert.Equal(t, "INTERNAL", valueobject.RailInternal.String())
}
