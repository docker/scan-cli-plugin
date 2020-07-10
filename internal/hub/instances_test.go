package hub

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestInstance_FetchJwks(t *testing.T) {
	instance := GetInstance()
	got, err := instance.FetchJwks()
	assert.NilError(t, err)
	assert.Assert(t, len(got.Keys) >= 1)
}
