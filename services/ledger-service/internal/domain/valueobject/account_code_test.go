package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func TestNewAccountCode_ValidCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{name: "four digits", code: "1000"},
		{name: "four digits with sub-account", code: "1000-001"},
		{name: "all nines", code: "9999"},
		{name: "all nines with sub-account", code: "9999-999"},
		{name: "all zeros", code: "0000"},
		{name: "all zeros with sub-account", code: "0000-000"},
		{name: "mixed digits", code: "2345-678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac, err := valueobject.NewAccountCode(tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.code, ac.Code())
			assert.Equal(t, tt.code, ac.String())
			assert.False(t, ac.IsZero())
		})
	}
}

func TestNewAccountCode_InvalidCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{name: "empty string", code: ""},
		{name: "alphabetic", code: "abcd"},
		{name: "too few digits", code: "100"},
		{name: "too many digits", code: "12345"},
		{name: "sub-account too short", code: "1000-01"},
		{name: "sub-account too long", code: "1000-0001"},
		{name: "missing sub-account digits", code: "1000-"},
		{name: "double dash", code: "1000--001"},
		{name: "spaces", code: "1000 001"},
		{name: "letters in code", code: "100A-001"},
		{name: "special characters", code: "1000/001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobject.NewAccountCode(tt.code)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid account code")
		})
	}
}

func TestAccountCode_IsZero(t *testing.T) {
	var zeroCode valueobject.AccountCode
	assert.True(t, zeroCode.IsZero())

	validCode, err := valueobject.NewAccountCode("1000")
	require.NoError(t, err)
	assert.False(t, validCode.IsZero())
}

func TestAccountCode_Equal(t *testing.T) {
	code1, err := valueobject.NewAccountCode("1000")
	require.NoError(t, err)

	code2, err := valueobject.NewAccountCode("1000")
	require.NoError(t, err)

	code3, err := valueobject.NewAccountCode("2000")
	require.NoError(t, err)

	assert.True(t, code1.Equal(code2))
	assert.False(t, code1.Equal(code3))
}

func TestMustAccountCode_Valid(t *testing.T) {
	assert.NotPanics(t, func() {
		ac := valueobject.MustAccountCode("1000-001")
		assert.Equal(t, "1000-001", ac.Code())
	})
}

func TestMustAccountCode_Invalid_Panics(t *testing.T) {
	assert.Panics(t, func() {
		valueobject.MustAccountCode("invalid")
	})
}

func TestMustAccountCode_Empty_Panics(t *testing.T) {
	assert.Panics(t, func() {
		valueobject.MustAccountCode("")
	})
}
