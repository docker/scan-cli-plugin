package provider

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsAuthenticationError(t *testing.T) {
	assert.Assert(t, IsAuthenticationError(&authenticationError{}))
	assert.Assert(t, !IsAuthenticationError(errors.New("")))
}

func TestIsInvalidTokenError(t *testing.T) {
	assert.Assert(t, IsInvalidTokenError(&invalidTokenError{}))
	assert.Assert(t, !IsInvalidTokenError(errors.New("")))
}
