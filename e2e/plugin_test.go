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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/stretchr/testify/require"

	"github.com/docker/scan-cli-plugin/config"
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

func TestFirstScanOptinMessage(t *testing.T) {
	t.Skip("Issue with to get the input send through the virtual console")
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	// create a scan config file with optin disabled
	createScanConfigFileOptin(t, configDir, false)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.Nil(t, err)
	defer console.Close() //nolint:errcheck

	// docker scan should ask for consent the first time the user runs it
	cmd.Command = dockerCli.Command("scan", "--version")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()

	go func() {
		_, err := console.ExpectEOF()
		assert.NilError(t, err)
	}()

	result := icmd.StartCmd(cmd)

	time.Sleep(1 * time.Second)
	_, err = console.Expect(expect.WithTimeout(1*time.Second),
		expect.String("Docker Scan relies upon access to Snyk a third party provider, do you consent to proceed using Snyk?"))
	assert.NilError(t, err)
	time.Sleep(1 * time.Second)
	_, err = console.SendLine("y")
	assert.NilError(t, err)
	time.Sleep(1 * time.Second)

	result = icmd.WaitOnCmd(time.Duration(0), result)

	_ = result.Assert(t, icmd.Expected{
		ExitCode: 1, // 1 is OK as docker scan just shows help and exit 1
	}).Combined()

	// check the consent has been stored in config file
	data, err := ioutil.ReadFile(filepath.Join(configDir, "scan", "config.json"))
	assert.NilError(t, err)
	var conf config.Config
	assert.NilError(t, json.Unmarshal(data, &conf))
	assert.Equal(t, conf.Optin, true)
}

func TestHandleCtrlCGracefully(t *testing.T) {
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
