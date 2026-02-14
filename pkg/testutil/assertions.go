package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RequireNoError fails the test immediately if err is not nil.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// AssertErrorContains checks that err contains the expected substring.
func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expected)
}
