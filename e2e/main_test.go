package e2e

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

var (
	dockerCli dockerCliCommand
)

type dockerCliCommand struct {
	path         string
	cliPluginDir string
}

func (d dockerCliCommand) createTestCmd() (icmd.Cmd, string, func()) {
	configDir, err := ioutil.TempDir("", "config")
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Join(configDir, "scan"), 0644); err != nil {
		panic(err)
	}
	createSymLink("snyk", "/root/.docker/scan", filepath.Join(configDir, "scan"))

	configFilePath := filepath.Join(configDir, "config.json")
	config := dockerConfigFile.ConfigFile{
		CLIPluginsExtraDirs: []string{
			d.cliPluginDir,
		},
		Filename: configFilePath,
	}
	configFile, err := os.Create(configFilePath)
	if err != nil {
		panic(err)
	}
	//nolint:errcheck
	defer configFile.Close()
	err = json.NewEncoder(configFile).Encode(config)
	if err != nil {
		panic(err)
	}
	cleanup := func() {
		_ = os.RemoveAll(configDir)
	}
	env := append(os.Environ(),
		"DOCKER_CONFIG="+configDir,
		"DOCKER_CLI_EXPERIMENTAL=enabled") // TODO: Remove this once docker app plugin is no more experimental
	return icmd.Cmd{Env: env}, configDir, cleanup
}

func (d dockerCliCommand) Command(args ...string) []string {
	return append([]string{d.path}, args...)
}

func TestMain(m *testing.M) {
	// Prepare docker cli to call the docker-app plugin binary:
	// - Create a symbolic link with the dockerApp binary to the plugin directory
	cliPluginDir, err := ioutil.TempDir("", "configContent")
	if err != nil {
		panic(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(cliPluginDir)
	createSymLink("docker-scan", "/root/.docker/cli-plugins", cliPluginDir)

	dockerCli = dockerCliCommand{path: "docker", cliPluginDir: cliPluginDir}
	os.Exit(m.Run())
}

func createSymLink(binaryName, sourceDir, configDir string) {
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	if err := os.Symlink(filepath.Join(sourceDir, binaryName), filepath.Join(configDir, binaryName)); err != nil {
		panic(err)
	}
}

func overloadEnvVariable(t *testing.T, envVar string, value string) func() {
	initialValue := os.Getenv(envVar)
	err := os.Setenv(envVar, value)
	assert.NilError(t, err)
	return  func() { _ = os.Setenv(envVar, initialValue) }
}
