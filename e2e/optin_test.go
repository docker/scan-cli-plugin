// +build !windows

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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/docker/scan-cli-plugin/config"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

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
		expect.String("Docker Scan relies upon access to Snyk, a third party provider, do you consent to proceed using Snyk?"))
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

func TestRefuseOptinWithDisableFlag(t *testing.T) {
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// docker scan --version should exit immediately
	cmd.Command = dockerCli.Command("scan", "--reject-license", "--version")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "",
	})

	// check the consent has been stored in config file
	data, err := ioutil.ReadFile(filepath.Join(configDir, "scan", "config.json"))
	assert.NilError(t, err)
	var conf config.Config
	assert.NilError(t, json.Unmarshal(data, &conf))
	assert.Equal(t, conf.Optin, false)
}
