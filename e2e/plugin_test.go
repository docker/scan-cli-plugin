package e2e

import (
	"regexp"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/golden"
	"gotest.tools/icmd"
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

	// docker info should print scan version and short description
	cmd.Command = dockerCli.Command("info")
	re := regexp.MustCompile(`scan: Docker Scan \(Docker Inc\., .*\)`)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Assert(t, re.MatchString(output))
}
