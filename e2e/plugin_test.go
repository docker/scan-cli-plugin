package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mitchellh/go-ps"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
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

func TestHandleCtrlCGracefully(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows platform does not handle SIGINT")
	}
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// Create a dummy snyk binary which takes long to exit
	assert.NilError(t, ioutil.WriteFile(filepath.Join(configDir, "scan", "snyk"), []byte(`#!/bin/sh
sleep 1000`), 0700))
	createScanConfigFile(t, configDir)

	// Add mock snyk binary to the $PATH
	path := os.Getenv("PATH")
	defer env.Patch(t, "PATH", fmt.Sprintf(pathFormat(), configDir+"/scan", path))()

	cmd.Command = dockerCli.Command("scan", "--version")
	icmd.StartCmd(cmd)
	time.Sleep(1 * time.Second)

	// Snyk command should be running
	scanProcess := shouldProcessBeRunning(t, "docker-scan", true, 0)
	sp, err := os.FindProcess(scanProcess.Pid())
	assert.NilError(t, err)

	shouldProcessBeRunning(t, "snyk", true, scanProcess.Pid())

	// send interrupt signal to the docker scan --version command
	assert.NilError(t, sp.Signal(syscall.SIGINT))
	time.Sleep(1 * time.Second)

	// Snyk command should be terminated too
	shouldProcessBeRunning(t, "snyk", false, scanProcess.Pid())
}

func shouldProcessBeRunning(t *testing.T, executable string, running bool, PPID int) ps.Process {
	t.Helper()
	processes, err := ps.Processes()
	assert.NilError(t, err)
	for _, process := range processes {
		if strings.HasPrefix(process.Executable(), executable) || (PPID != 0 && process.PPid() == PPID) {
			assert.Assert(t, process.Pid() != 0)
			if PPID != 0 {
				assert.Equal(t, process.PPid(), PPID) // snyk is the child process of docker scan
			}
			assert.Assert(t, running)
			return process
		}
	}
	assert.Assert(t, !running)
	return nil
}
