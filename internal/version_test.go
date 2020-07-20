/*
   Copyright 2020 Docker Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
