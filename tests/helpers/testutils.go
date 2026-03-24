// tests/helpers/testutils.go
package helpers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func AssertNoError(t *testing.T, err error) {
	require.NoError(t, err)
}

func AssertError(t *testing.T, err error, msg string) {
	require.Error(t, err)
	assert.Contains(t, err.Error(), msg)
}
