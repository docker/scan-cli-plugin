package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

func TestScanFailsNoAuthentication(t *testing.T) {
	// create Snyk config file with empty token
	homeDir := createSnykConfFile(t, "")
	defer homeDir.Remove()
	defer overloadEnvVariable(t, "HOME", homeDir.Path()) ()

	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// write dockerCli config with authentication to a registry which isn't Hub
	patchConfig(t, configDir, "com.example.registry")

	cmd.Command = dockerCli.Command("scan", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err: `You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`,
	})
}

func TestScanFailsWithCleanMessage(t *testing.T) {
	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("scan", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err: `You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`,
	})
}

func TestScanSucceedWithDockerHub(t *testing.T) {
	cmd, configDir, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// write dockerCli config with authentication to a registry which isn't Hub
	patchConfig(t, configDir, "https://index.docker.io/v1/")

	cmd.Command = dockerCli.Command("scan", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func TestScanSucceedWithSnyk(t *testing.T) {
	// create Snyk config file with empty token
	homeDir := createSnykConfFile(t, "valid-token")
	defer homeDir.Remove()
	defer overloadEnvVariable(t, "HOME", homeDir.Path()) ()

	cmd, _, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("scan", "example:image")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func createSnykConfFile(t *testing.T, token string) *fs.Dir {
	content := fmt.Sprintf(`{"api" : "%s"}`, token)
	return fs.NewDir(t, t.Name(),
		fs.WithDir(".config",
			fs.WithDir("configstore",
				fs.WithFile("snyk.json", content))))
}

func patchConfig(t *testing.T, configDir string, url string) {
	buff, err := ioutil.ReadFile(filepath.Join(configDir, "config.json"))
	assert.NilError(t, err)
	var conf configfile.ConfigFile
	assert.NilError(t, json.Unmarshal(buff, &conf))

	conf.AuthConfigs = map[string]types.AuthConfig{url: {}}
	buff, err = json.Marshal(&conf)
	assert.NilError(t, err)

	assert.NilError(t, ioutil.WriteFile(filepath.Join(configDir, "config.json"), buff, 0644))
}
