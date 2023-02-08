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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/scan-cli-plugin/config"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

const (
	ImageWithVulnerabilities      = "alpine:3.10.0"
	ImageWithoutVulnerabilities   = "hello-world"
	InvalidImage                  = "dockerscanci/scratch:1.0"          // FROM scratch
	ImageBaseImageVulnerabilities = "dockerscanci/base-image-vulns:1.0" // FROM alpine:3.10.0
	LocalBuildImage               = "local:build"
)

func TestScanFailsNoAuthentication(t *testing.T) {
	// create Snyk config file with empty token
	_, cleanFunction := createSnykConfFile(t, "")
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	createScanConfigFile(t, configDir)

	// write dockerCli config with authentication to a registry which isn't Hub
	patchConfig(t, configDir, "com.example.registry", "invalid-user", "invalid-password")

	cmd.Command = dockerCli.Command("scan", "--accept-license", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `failed to get DockerScanID: You need to be logged in to Docker Hub to use the scan feature.`,
	})
}

func TestScanFailsWithCleanMessage(t *testing.T) {
	// create Snyk config file with empty token
	_, cleanFunction := createSnykConfFile(t, "")
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("logout")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `failed to get DockerScanID: You need to be logged in to Docker Hub to use the scan feature.`,
	})
}

func TestScanSucceedWithDockerHub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Can't run on this ci platform (image does not exist fir the current platform)")
	}
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	// write dockerCli config with authentication to the Hub
	patchConfig(t, configDir, os.Getenv("E2E_HUB_URL"), os.Getenv("E2E_HUB_USERNAME"), os.Getenv("E2E_HUB_TOKEN"))

	cmd.Command = dockerCli.Command("scan", ImageWithVulnerabilities)
	result := icmd.RunCmd(cmd)
	assert.Assert(t, result.ExitCode == 1)
	if strings.HasPrefix(result.Combined(), "You") {
		// We reach the monthly limits of 10 free scans
		assert.Assert(t, strings.Contains(result.Combined(), "You have reached the scan limit of 10 monthly scans without authentication."), result.Combined())
	} else {
		assert.Assert(t, cmp.Regexp("found .* vulnerabilities", result.Combined()), result.Combined())

	}

}

func TestScanWithSnyk(t *testing.T) {
	if runtime.GOOS != "linux" {
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
		// Due to an issue linked to github actions env, we removed the test for the moment
		// we got the error message that Snyk returns when it can't connect to the engine 'Invalid Docker archive'
		/*{
			name:     "invalid-docker-archive",
			image:    InvalidImage,
			exitCode: 1,
			contains: "(HTTP code 500) server error - empty export - not implemented",
		},*/
		{
			name:     "image-with-vulnerabilities",
			image:    ImageWithVulnerabilities,
			exitCode: 1,
			contains: "vulnerability found",
		},
		{
			name:     "invalid-image-name",
			image:    "scratch",
			exitCode: 1,
			contains: "manifest unknown",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd.Command = dockerCli.Command("scan", testCase.image)
			output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: testCase.exitCode}).Combined()
			assert.Assert(t, strings.Contains(output, testCase.contains), output)
		})
	}
}

func TestScanJsonOutput(t *testing.T) {
	if runtime.GOOS != "linux" {
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
			name:     "invalid-docker-archive",
			image:    InvalidImage,
			exitCode: 1,
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
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--file", "./testdata/Dockerfile", "--exclude-base", ImageBaseImageVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Assert(t, strings.Contains(output, "no vulnerable paths found."))
}

func TestScanWithExcludeBaseImageVulns(t *testing.T) {
	if runtime.GOOS != "linux" {
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
	if runtime.GOOS != "linux" {
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

func TestScanWithSeverity(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--severity=medium", ImageWithVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(output, "alpine:3.10.0")) // beginning of the dependency tree
	assert.Assert(t, cmp.Regexp("found .* issues", output))
	assert.Assert(t, !strings.Contains(output, "Low severity"))
}

func TestScanWithSeverityBadValue(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--severity=unsupportedValue", ImageWithVulnerabilities)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "--severity takes only 'low', 'medium' or 'high' values"})
}

func TestScanWithJsonAndGroupIssues(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--json", "--group-issues", ImageWithVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(output, "vulnerable dependency paths")) // beginning of the dependency tree
	assert.Assert(t, strings.Contains(output, `"from": [
        [
          "docker-image|alpine@3.10.0",`))
}

func TestScanWithGroupIssues(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	createScanConfigFile(t, configDir)

	cmd.Command = dockerCli.Command("scan", "--accept-license", "--group-issues", ImageBaseImageVulnerabilities)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "--json flag is mandatory to use --group-issues flag"})
}

func TestScanWithContainerizedSnyk(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	homeDir, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	createScanConfigFileOptinAndPath(t, configDir, true, "")

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
			name:     "invalid-docker-archive",
			image:    InvalidImage,
			exitCode: 1,
			contains: "(HTTP code 500) server error - empty export - not implemented",
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
			exitCode: 1,
			contains: "manifest unknown",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd.Command = dockerCli.Command("scan", testCase.image)
			cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", homeDir.Path()))
			output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: testCase.exitCode}).Combined()
			assert.Assert(t, strings.Contains(output, testCase.contains), output)
		})
	}
}

func TestScanLocalImageWithContainerizedSnyk(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	createScanConfigFileOptinAndPath(t, configDir, true, "")

	// Build a local image
	cmd.Command = dockerCli.Command("build", "-f", "./testdata/Dockerfile", "-t", LocalBuildImage, ".")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("scan", LocalBuildImage)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(output, "vulnerability found"))
}

func TestScanWithFileAndExcludeBaseImageVulnsContainerizedProvider(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Can't run on this ci platform (windows containers or no engine installed)")
	}
	pwd, _ := os.Getwd()
	dockerfilePath := path.Join(pwd, "/testdata/Dockerfile")
	_, cleanFunction := createSnykConfFile(t, os.Getenv("E2E_TEST_AUTH_TOKEN"))
	defer cleanFunction()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	createScanConfigFileOptinAndPath(t, configDir, true, "")

	cmd.Command = dockerCli.Command("scan", "--file", dockerfilePath, "--exclude-base", ImageBaseImageVulnerabilities)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Assert(t, strings.Contains(output, "no vulnerable paths found."))
}

func createSnykConfDirectories(t *testing.T, withConfFile bool, token string) (*fs.Dir, func()) {
	content := fmt.Sprintf(`{"api" : "%s"}`, token)
	var confFiles []fs.PathOp
	if withConfFile {
		confFiles = append(confFiles, fs.WithFile("snyk.json", content))
	}
	homeDir := fs.NewDir(t, t.Name(),
		fs.WithDir(".config",
			fs.WithDir("configstore", confFiles...)))

	homeFunc := env.Patch(t, "HOME", homeDir.Path())
	userProfileFunc := env.Patch(t, "USERPROFILE", homeDir.Path())
	cleanup := func() {
		userProfileFunc()
		homeFunc()
		homeDir.Remove()
	}

	return homeDir, cleanup
}

func createSnykConfFile(t *testing.T, token string) (*fs.Dir, func()) {
	return createSnykConfDirectories(t, true, token)
}

func patchConfig(t *testing.T, configDir, url, userName, password string) {
	buff, err := os.ReadFile(filepath.Join(configDir, "config.json"))
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

	assert.NilError(t, os.WriteFile(filepath.Join(configDir, "config.json"), buff, 0644))
}

func createScanConfigFile(t *testing.T, configDir string) {
	createScanConfigFileOptin(t, configDir, true)
}

func createScanConfigFileOptin(t *testing.T, configDir string, optin bool) {
	createScanConfigFileOptinAndPath(t, configDir, optin, filepath.Join(configDir, "scan", "snyk"))
}

func createScanConfigFileOptinAndPath(t *testing.T, configDir string, optin bool, path string) {
	conf := config.Config{
		Path:  path,
		Optin: optin,
	}
	buf, err := json.MarshalIndent(conf, "", "  ")
	assert.NilError(t, err)
	err = os.WriteFile(filepath.Join(configDir, "scan", "config.json"), buf, 0644)
	assert.NilError(t, err)
}
