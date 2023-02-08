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
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// Create a dummy snyk binary which takes long to exit
	assert.NilError(t, os.WriteFile(filepath.Join(configDir, "scan", "snyk"), []byte(`#!/bin/sh
sleep 1000`), 0700))
	createScanConfigFile(t, configDir)

	// Add mock snyk binary to the $PATH
	path := os.Getenv("PATH")
	path = fmt.Sprintf(pathFormat(), configDir+"/scan", path)
	env.Patch(t, "PATH", path)()
	// force the env variable on command side on Windows
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path))
	}

	cmd.Command = dockerCli.Command("scan", "--version")

	icmd.StartCmd(cmd)
	time.Sleep(1 * time.Second)

	// Snyk command should be running
	scanProcess := shouldProcessBeRunning(t, "docker-scan", true, 0)
	sp, err := os.FindProcess(scanProcess.Pid())
	assert.NilError(t, err)

	shouldProcessBeRunning(t, "snyk", true, scanProcess.Pid())

	// send interrupt signal to the docker scan --version command
	switch runtime.GOOS {
	case "windows":
		assert.NilError(t, sp.Kill()) // windows can't handle SIGINT
	case "darwin", "linux":
		assert.NilError(t, sp.Signal(syscall.SIGINT))
	default:
		t.Fatalf("os %q not supported", runtime.GOOS)
	}
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
