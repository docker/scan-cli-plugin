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

package provider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

var (
	_ Provider = &snykProvider{}
)

const (
	snykToken = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
)

func TestSnykLoginEnvVars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Can't run this test on windows")
	}

	provider, outStream := setupMockSnykBinary(t)

	err := provider.Authenticate(snykToken)
	assert.NilError(t, err)

	// SNYK_INTEGRATION is always set
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_INTEGRATION_NAME=DOCKER_DESKTOP"))
	// NO_UPDATE_NOTIFIER disables node.js automatic update notification in console
	assert.Assert(t, strings.Contains(outStream.String(), "NO_UPDATE_NOTIFIER=true"))
	// SNYK_CFG_DISABLESUGGESTIONS removes user hints from snyk
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_CFG_DISABLESUGGESTIONS=true"))
	// Check UTMs variables
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_UTM_MEDIUM=Partner"))
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_UTM_SOURCE=Docker"))
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_UTM_CAMPAIGN=Docker-Desktop-2020"))
}

func TestSnykScanEnvVars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Can't run this test on windows")
	}

	// Create Snyk config file with token
	tempDir := fs.NewDir(t, t.Name(),
		fs.WithDir(".config",
			fs.WithDir("configstore",
				fs.WithFile("snyk.json", `{"api":"`+snykToken+`"}`))))
	defer tempDir.Remove()
	defer env.Patch(t, "HOME", tempDir.Path())()

	provider, outStream := setupMockSnykBinary(t)

	err := provider.Scan("image")
	assert.NilError(t, err)

	// SNYK_INTEGRATION is always set
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_INTEGRATION_NAME=DOCKER_DESKTOP"))
	// NO_UPDATE_NOTIFIER disables node.js automatic update notification in console
	assert.Assert(t, strings.Contains(outStream.String(), "NO_UPDATE_NOTIFIER=true"))
	// SNYK_CFG_DISABLESUGGESTIONS removes user hints from snyk
	assert.Assert(t, strings.Contains(outStream.String(), "SNYK_CFG_DISABLESUGGESTIONS=true"))
}

func setupMockSnykBinary(t *testing.T) (Provider, *bytes.Buffer) {
	pwd, err := os.Getwd()
	assert.NilError(t, err)
	snykPath := filepath.Join(pwd, "testdata", "snyk")
	outStream := bytes.NewBuffer(nil)
	errStream := bytes.NewBuffer(nil)

	defaultProvider, err := NewProvider(WithContext(context.Background()),
		WithStreams(outStream, errStream))
	assert.NilError(t, err)
	provider, err := NewSnykProvider(
		defaultProvider,
		WithPath(snykPath))
	assert.NilError(t, err)
	return provider, outStream
}
