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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/icmd"
)

func TestSnykAuthentication(t *testing.T) {
	// Add snyk binary to the path
	path := os.Getenv("PATH")
	defer env.Patch(t, "PATH", fmt.Sprintf(pathFormat(), os.Getenv("SNYK_USER_PATH"), path))()

	// create Snyk config file with empty token
	homeDir, cleanFunction := createSnykConfFile(t, "")
	defer cleanFunction()

	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	token := os.Getenv("E2E_TEST_AUTH_TOKEN")
	assert.Assert(t, token != "", "E2E_TEST_AUTH_TOKEN needs to be filled")

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--login", "--token", token)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// snyk config file should be updated
	buff, err := ioutil.ReadFile(homeDir.Join(".config", "configstore", "snyk.json"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(buff), token))
}

func TestAuthenticationFlagFailsWithImage(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	token := os.Getenv("E2E_TEST_AUTH_TOKEN")
	assert.Assert(t, token != "", "E2E_TEST_AUTH_TOKEN needs to be filled")

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--login", "--token", token, "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "--login flag expects no argument",
	})
}

func TestAuthenticationChecksToken(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--login", "--token", "invalid-token")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `invalid authentication token "invalid-token"`,
	})
}
