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

package e2e

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/icmd"

	"github.com/docker/scan-cli-plugin/internal"
)

func TestVersionSnykUserBinary(t *testing.T) {
	// Add user snyk binary to the $PATH
	path := os.Getenv("PATH")
	defer env.Patch(t, "PATH", fmt.Sprintf(pathFormat(), os.Getenv("SNYK_USER_PATH"), path))()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	// docker scan --version should use user's Snyk binary
	cmd.Command = dockerCli.Command("scan", "--version")
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	expected := fmt.Sprintf(
		`Version:    %s
Git commit: %s
Provider:   %s
`, internal.Version, internal.GitCommit, getProviderVersion("SNYK_USER_VERSION"))

	assert.Equal(t, output, expected)
}

func TestVersionSnykDesktopBinary(t *testing.T) {
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	// docker scan --version should print docker scan plugin version and snyk version
	cmd.Command = dockerCli.Command("scan", "--version")
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	expected := fmt.Sprintf(
		`Version:    %s
Git commit: %s
Provider:   %s
`, internal.Version, internal.GitCommit, getProviderVersion("SNYK_DESKTOP_VERSION"))

	assert.Equal(t, output, expected)
}

func TestVersionWithoutSnykOrConfig(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// docker scan --version should fail with a clean error
	cmd.Command = dockerCli.Command("scan", "--version")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "could not find Snyk binary",
	})
}

func getProviderVersion(env string) string {
	return fmt.Sprintf("Snyk (%s (standalone))", os.Getenv(env))
}
