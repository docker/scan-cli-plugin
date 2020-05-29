package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker-scan/internal"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestVersion(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

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
	return fmt.Sprintf("Snyk (%s)", os.Getenv("SNYK_VERSION"))
}
