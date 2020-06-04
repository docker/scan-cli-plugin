package e2e

import (
	"testing"

	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestInvokePluginFromCLI(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	// docker --help should list app as a top command
	cmd.Command = dockerCli.Command("--help")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		Out: "scan*       Docker Scan (Docker Inc.",
	})

	// docker app --help prints docker-app help
	cmd.Command = dockerCli.Command("scan", "--help")
	usage := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()

	goldenFile := "plugin-usage.golden"
	golden.Assert(t, usage, goldenFile)
}
