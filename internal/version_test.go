package internal

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

type providerStub struct {
	version string
}

func (s *providerStub) Version() (string, error) {
	return s.version, nil
}

func (s *providerStub) Scan(image string) error {
	return nil
}

func (s *providerStub) Authenticate(token string) error {
	return nil
}

func TestFullVersion(t *testing.T) {
	stub := &providerStub{version: "stub-version"}
	actual, err := FullVersion(stub)
	assert.NilError(t, err)
	expected := fmt.Sprintf(
		`Version:    %s
Git commit: %s
Provider:   %s`, Version, GitCommit, "stub-version")
	assert.Equal(t, actual, expected)
}
