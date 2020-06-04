package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/docker-scan/config"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"

	"github.com/docker/docker-scan/internal"
)

func TestVersionSnykUserBinary(t *testing.T) {
	// Add user snyk binary to the $PATH
	path := os.Getenv("PATH")
	defer overloadEnvVariable(t, "PATH", fmt.Sprintf("%s:%s", "/root/e2e", path))()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// Create a config file pointing to desktop snyk binary
	conf := config.Config{Path: fmt.Sprintf("%s/scan/snyk", configDir)}
	buf, err := json.MarshalIndent(conf, "", "  ")
	assert.NilError(t, err)
	err = ioutil.WriteFile(fmt.Sprintf("%s/scan/config.json", configDir), buf, 0644)
	assert.NilError(t, err)

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

	// Create a config file pointing to desktop snyk binary
	conf := config.Config{Path: fmt.Sprintf("%s/scan/snyk", configDir)}
	buf, err := json.MarshalIndent(conf, "", "  ")
	assert.NilError(t, err)
	err = ioutil.WriteFile(fmt.Sprintf("%s/scan/config.json", configDir), buf, 0644)
	assert.NilError(t, err)

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
