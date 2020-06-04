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

	cmd.Command = dockerCli.Command("scan", "--auth", token)
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

	cmd.Command = dockerCli.Command("scan", "--auth", token, "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "--auth flag expects maximum one argument",
	})
}

func TestAuthenticationChecksToken(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("scan", "--auth", "invalid-token")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `invalid authentication token "invalid-token"`,
	})
}
