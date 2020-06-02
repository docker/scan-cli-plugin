package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/docker-scan/config"

	"github.com/docker/docker-scan/internal"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestVersionSnykDesktopBinary(t *testing.T) {
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// Create a config file pointing to desktop snyk binary
	conf := config.Config{Path: fmt.Sprintf("%s/scan/snyk", configDir)}
	buf, err := json.MarshalIndent(conf, "", "  ")
	assert.NilError(t, err)
	err = ioutil.WriteFile(fmt.Sprintf("%s/scan/config.json", configDir), buf, 0644)
	assert.NilError(t, err)

	// docker --help should list app as a top command
	cmd.Command = dockerCli.Command("scan", "--version")
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	expected := fmt.Sprintf(
		`Version:    %s
Git commit: %s
Provider:   %s
`, internal.Version, internal.GitCommit, getProviderVersion())

	assert.Equal(t, output, expected)
}

func getProviderVersion() string {
	return fmt.Sprintf("Snyk (%s (standalone))", os.Getenv("SNYK_DESKTOP_VERSION"))
}
