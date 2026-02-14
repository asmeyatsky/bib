package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

func TestNewVerificationStatus_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected valueobject.VerificationStatus
	}{
		{"PENDING", valueobject.StatusPending},
		{"IN_PROGRESS", valueobject.StatusInProgress},
		{"APPROVED", valueobject.StatusApproved},
		{"REJECTED", valueobject.StatusRejected},
		{"EXPIRED", valueobject.StatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			status, err := valueobject.NewVerificationStatus(tt.input)
			require.NoError(t, err)
			assert.True(t, status.Equal(tt.expected))
			assert.Equal(t, tt.input, status.String())
		})
	}
}

func TestNewVerificationStatus_InvalidValue(t *testing.T) {
	tests := []string{
		"",
		"UNKNOWN",
		"pending",
		"approved",
		"INVALID",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := valueobject.NewVerificationStatus(input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown verification status")
		})
	}
}

func TestVerificationStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status     valueobject.VerificationStatus
		isTerminal bool
	}{
		{valueobject.StatusPending, false},
		{valueobject.StatusInProgress, false},
		{valueobject.StatusApproved, true},
		{valueobject.StatusRejected, true},
		{valueobject.StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.isTerminal, tt.status.IsTerminal())
		})
	}
}

func TestVerificationStatus_String(t *testing.T) {
	assert.Equal(t, "PENDING", valueobject.StatusPending.String())
	assert.Equal(t, "IN_PROGRESS", valueobject.StatusInProgress.String())
	assert.Equal(t, "APPROVED", valueobject.StatusApproved.String())
	assert.Equal(t, "REJECTED", valueobject.StatusRejected.String())
	assert.Equal(t, "EXPIRED", valueobject.StatusExpired.String())
}

func TestVerificationStatus_Equal(t *testing.T) {
	s1, _ := valueobject.NewVerificationStatus("PENDING")
	s2, _ := valueobject.NewVerificationStatus("PENDING")
	s3, _ := valueobject.NewVerificationStatus("APPROVED")

	assert.True(t, s1.Equal(s2))
	assert.False(t, s1.Equal(s3))
}

func TestNewCheckType_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected valueobject.CheckType
	}{
		{"DOCUMENT", valueobject.CheckTypeDocument},
		{"SELFIE", valueobject.CheckTypeSelfie},
		{"WATCHLIST", valueobject.CheckTypeWatchlist},
		{"ADDRESS", valueobject.CheckTypeAddress},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ct, err := valueobject.NewCheckType(tt.input)
			require.NoError(t, err)
			assert.True(t, ct.Equal(tt.expected))
			assert.Equal(t, tt.input, ct.String())
		})
	}
}

func TestNewCheckType_InvalidValue(t *testing.T) {
	tests := []string{
		"",
		"UNKNOWN",
		"document",
		"BIOMETRIC",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := valueobject.NewCheckType(input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown check type")
		})
	}
}

func TestDefaultCheckTypes(t *testing.T) {
	defaults := valueobject.DefaultCheckTypes()
	require.Len(t, defaults, 3)
	assert.True(t, defaults[0].Equal(valueobject.CheckTypeDocument))
	assert.True(t, defaults[1].Equal(valueobject.CheckTypeSelfie))
	assert.True(t, defaults[2].Equal(valueobject.CheckTypeWatchlist))
}
