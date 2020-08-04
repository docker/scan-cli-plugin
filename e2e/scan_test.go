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
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/scan-cli-plugin/config"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

const (
	ImageWithVulnerabilities      = "alpine:3.10.0"
	ImageWithoutVulnerabilities   = "dockerscanci/scratch:1.0"          // FROM scratch
	ImageBaseImageVulnerabilities = "dockerscanci/base-image-vulns:1.0" // FROM alpine:3.10.0
)

func TestScanFailsNoAuthentication(t *testing.T) {
	// create Snyk config file with empty token
	_, cleanFunction := createSnykConfFile(t, "")
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// write dockerCli config with authentication to a registry which isn't Hub
	patchConfig(t, configDir, "com.example.registry", "invalid-user", "invalid-password")

	cmd.Command = dockerCli.Command("scan", "--accept-license", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err: `You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`,
	})
}

func TestScanFailsWithCleanMessage(t *testing.T) {
	// create Snyk config file with empty token
	_, cleanFunction := createSnykConfFile(t, "")
	defer cleanFunction()

	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("scan", "--accept-license", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err: `You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`,
	})
}

func TestScanSucceedWithDockerHub(t *testing.T) {
	t.Skip("TODO: waiting for Hub ID generation")
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	// write dockerCli config with authentication to the Hub
	patchConfig(t, configDir, os.Getenv("E2E_HUB_URL"), os.Getenv("E2E_HUB_USERNAME"), os.Getenv("E2E_HUB_TOKEN"))

	cmd.Command = dockerCli.Command("scan", ImageWithVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(output, "vulnerability found"))

	// Check that token file has been created
	buf, err := ioutil.ReadFile(filepath.Join(configDir, "scan", "--accept-license", "scan-id.json"))
	assert.NilError(t, err)
	var scanID struct {
		Identifier string `json:"id"`
	}
	assert.NilError(t, json.Unmarshal(buf, &scanID))
	assert.Equal(t, len(strings.Split(scanID.Identifier, ".")), 3)
}

func TestScanWithSnyk(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	testCases := []struct {
		name     string
		image    string
		exitCode int
		contains string
	}{
		{
			name:     "image-without-vulnerabilities",
			image:    ImageWithoutVulnerabilities,
			exitCode: 0,
			contains: "no vulnerable paths found",
		},
		{
			name:     "image-with-vulnerabilities",
			image:    ImageWithVulnerabilities,
			exitCode: 1,
			contains: "vulnerability found",
		},
		{
			name:     "invalid-image-name",
			image:    "scratch",
			exitCode: 2,
			contains: "image was not found locally and pulling failed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd.Command = dockerCli.Command("scan", testCase.image)
			output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: testCase.exitCode}).Combined()
			assert.Assert(t, strings.Contains(output, testCase.contains))
		})
	}
}

func TestScanJsonOutput(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	testCases := []struct {
		name     string
		image    string
		exitCode int
		isEmpty  bool
	}{
		{
			name:     "image-without-vulnerabilities",
			image:    ImageWithoutVulnerabilities,
			exitCode: 0,
			isEmpty:  true,
		},
		{
			name:     "image-with-vulnerabilities",
			image:    ImageWithVulnerabilities,
			exitCode: 1,
			isEmpty:  false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd.Command = dockerCli.Command("scan", "--accept-license", "--json", testCase.image)
			output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: testCase.exitCode}).Combined()
			var jsonOutput JSONOutput
			assert.NilError(t, json.Unmarshal([]byte(output), &jsonOutput))
			assert.Equal(t, len(jsonOutput.Vulnerabilities) == 0, testCase.isEmpty)
		})
	}
}

type JSONOutput struct {
	Vulnerabilities []interface{} `json:"vulnerabilities"`
}

func TestScanWithFileAndExcludeBaseImageVulns(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--file", "./testdata/Dockerfile", "--exclude-base", ImageBaseImageVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Assert(t, strings.Contains(output, "found 0 issues."))
}

func TestScanWithExcludeBaseImageVulns(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--exclude-base", ImageBaseImageVulnerabilities)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "--file flag is mandatory to use --exclude-base flag"})
}

func TestScanWithDependencies(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--dependency-tree", ImageWithVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(output, "docker-image|alpine @ 3.10.0")) // beginning of the dependency tree
	assert.Assert(t, strings.Contains(output, "vulnerability found"))
}

func createSnykConfFile(t *testing.T, token string) (*fs.Dir, func()) {
	content := fmt.Sprintf(`{"api" : "%s"}`, token)
	homeDir := fs.NewDir(t, t.Name(),
		fs.WithDir(".config",
			fs.WithDir("configstore",
				fs.WithFile("snyk.json", content))))
	homeFunc := env.Patch(t, "HOME", homeDir.Path())
	userProfileFunc := env.Patch(t, "USERPROFILE", homeDir.Path())
	cleanup := func() {
		userProfileFunc()
		homeFunc()
		homeDir.Remove()
	}

	return homeDir, cleanup
}

func patchConfig(t *testing.T, configDir, url, userName, password string) {
	buff, err := ioutil.ReadFile(filepath.Join(configDir, "config.json"))
	assert.NilError(t, err)
	var conf configfile.ConfigFile
	assert.NilError(t, json.Unmarshal(buff, &conf))

	conf.AuthConfigs = map[string]types.AuthConfig{
		url: {
			Username: userName,
			Password: password,
		},
	}
	buff, err = json.Marshal(&conf)
	assert.NilError(t, err)

	assert.NilError(t, ioutil.WriteFile(filepath.Join(configDir, "config.json"), buff, 0644))
}

func createScanConfigFile(t *testing.T, configDir string) {
	createScanConfigFileOptin(t, configDir, true)
}

func createScanConfigFileOptin(t *testing.T, configDir string, optin bool) {
	conf := config.Config{
		Path:  filepath.Join(configDir, "scan", "snyk"),
		Optin: optin,
	}
	buf, err := json.MarshalIndent(conf, "", "  ")
	assert.NilError(t, err)
	err = ioutil.WriteFile(filepath.Join(configDir, "scan", "config.json"), buf, 0644)
	assert.NilError(t, err)
}
