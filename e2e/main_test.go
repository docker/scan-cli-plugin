package e2e

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
	"gotest.tools/icmd"
)

var (
	dockerCli dockerCliCommand
)

type dockerCliCommand struct {
	path         string
	cliPluginDir string
}

type ConfigFileOperator func(configFile *dockerConfigFile.ConfigFile)

func (d dockerCliCommand) createTestCmd(ops ...ConfigFileOperator) (icmd.Cmd, func()) {
	configDir, err := ioutil.TempDir("", "config")
	if err != nil {
		panic(err)
	}
	configFilePath := filepath.Join(configDir, "config.json")
	config := dockerConfigFile.ConfigFile{
		CLIPluginsExtraDirs: []string{
			d.cliPluginDir,
		},
		Filename: configFilePath,
	}
	for _, op := range ops {
		op(&config)
	}
	configFile, err := os.Create(configFilePath)
	if err != nil {
		panic(err)
	}
	defer configFile.Close()
	err = json.NewEncoder(configFile).Encode(config)
	if err != nil {
		panic(err)
	}
	cleanup := func() {
		os.RemoveAll(configDir)
	}
	env := append(os.Environ(),
		"DOCKER_CONFIG="+configDir,
		"DOCKER_CLI_EXPERIMENTAL=enabled") // TODO: Remove this once docker app plugin is no more experimental
	return icmd.Cmd{Env: env}, cleanup
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
	defer os.RemoveAll(cliPluginDir)
	createDockerAppSymLink("/root/.docker/cli-plugins/docker-scan", cliPluginDir)

	dockerCli = dockerCliCommand{path: "docker", cliPluginDir: cliPluginDir}
	os.Exit(m.Run())
}

func createDockerAppSymLink(dockerScan, configDir string) {
	dockerScanExecName := "docker-scan"
	if runtime.GOOS == "windows" {
		dockerScanExecName += ".exe"
	}
	if err := os.Symlink(dockerScan, filepath.Join(configDir, dockerScanExecName)); err != nil {
		panic(err)
	}
}
