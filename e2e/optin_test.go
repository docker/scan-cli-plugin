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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/scan-cli-plugin/config"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

func TestFirstScanOptinMessage(t *testing.T) {
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	// create a scan config file with optin disabled
	createScanConfigFileOptin(t, configDir, false)

	// docker scan should ask for consent the first time the user runs it
	in, out, err := os.Pipe()
	assert.NilError(t, err)
	cmd.Command = dockerCli.Command("scan", "myimage")
	cmd.Stdin = in

	go func() {
		time.Sleep(20 * time.Millisecond)
		fmt.Fprintln(out, "y")
	}()

	result := icmd.RunCmd(cmd)
	assert.Assert(t, strings.Contains(result.Combined(), `Docker Scan relies upon access to Snyk, a third party provider, do you consent to proceed using Snyk? (y/N)`))

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
	createScanConfigFile(t, configDir)

	// docker scan myimage should exit immediately
	cmd.Command = dockerCli.Command("scan", "--reject-license", "myimage")
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
