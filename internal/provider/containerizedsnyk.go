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

package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

// ImageDigest is the sha snyk/snyk:alpine image, set at build time
var (
	ImageDigest = "unknown"
	image       = fmt.Sprintf("snyk/snyk@%s", ImageDigest)
)

type dockerSnykProvider struct {
	cli command.Cli
	Options
}

type dockerEnvs []string
type dockerBindings []string

type removeContainerFunc func() error

func (d *dockerSnykProvider) removeContainer(containerID string) removeContainerFunc {
	return func() error {
		return d.cli.Client().ContainerRemove(d.context, containerID, types.ContainerRemoveOptions{})
	}
}

// NewDockerSnykProvider returns a containerized Snyk implementation of scan provider
func NewDockerSnykProvider(cli command.Cli, defaultProvider Options) (Provider, error) {
	provider := dockerSnykProvider{
		cli:     cli,
		Options: defaultProvider,
	}
	options := types.ImagePullOptions{}
	_, _, err := cli.Client().ImageInspectWithRaw(provider.context, image)
	if err != nil {
		responseBody, err := cli.Client().ImagePull(provider.context, image, options)
		if err != nil {
			return nil, err
		}
		//nolint: errcheck
		defer responseBody.Close()
		return &provider, jsonmessage.DisplayJSONMessagesStream(responseBody, bytes.NewBuffer(nil), cli.Out().FD(), false, nil)
	}
	return &provider, nil
}

// DockerSnykProviderOps function taking a pointer to a containerized Snyk Provider and returning an error if needed
type DockerSnykProviderOps func(provider *dockerSnykProvider) error

func (d *dockerSnykProvider) Authenticate(token string) error {
	if token != "" {
		if _, err := uuid.Parse(token); err != nil {
			return &invalidTokenError{token}
		}
	}
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	containerName := fmt.Sprintf("synk-auth-%s", uuid.New().String())

	containerID, removeContainer, err := d.createContainer(token, containerName)
	if err != nil {
		return err
	}
	//nolint: errcheck
	defer removeContainer()

	err = d.copySnykConfigToContainer(containerID, home)
	if err != nil {
		return err
	}

	streamFunc, err := d.startContainer(containerID)
	if err != nil {
		return err
	}
	err = d.checkContainerState(containerID)
	if err != nil {
		return err
	}
	streamFunc()
	return d.copySnykConfigToHost(containerName, home)
}

func (d *dockerSnykProvider) checkContainerState(containerID string) error {
	statusc, errc := d.cli.Client().ContainerWait(d.context, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errc:
		if err != nil {
			return err
		}
	case s := <-statusc:
		switch s.StatusCode {
		case 0:
		default:
			return containerizedError{}
		}
	}
	return nil
}

func (d *dockerSnykProvider) createContainer(token string, containerName string) (string, removeContainerFunc, error) {

	envVars := dockerEnvs{
		"NO_UPDATE_NOTIFIER=true",
		"SNYK_CFG_DISABLESUGGESTIONS=true",
		"SNYK_INTEGRATION_NAME=DOCKER_DESKTOP",
		"SNYK_UTM_MEDIUM=Partner",
		"SNYK_UTM_SOURCE=Docker",
		"SNYK_UTM_CAMPAIGN=Docker-Desktop-2020",
	}
	bindings := dockerBindings{
		"/var/run/docker.sock:/var/run/docker.sock",
		"TMP:/root/.config/configstore",
	}

	config, hostConfig := containerConfigs(envVars, bindings, strslice.StrSlice{"snyk", "auth", token})

	result, err := d.cli.Client().ContainerCreate(d.context, &config, &hostConfig, nil, &v1.Platform{Architecture: "amd64", OS: "linux"}, containerName)
	if err != nil {
		return "", nil, fmt.Errorf("cannot create container: %s", err)
	}
	removeContainerFunc := d.removeContainer(result.ID)
	return result.ID, removeContainerFunc, nil
}

func (d *dockerSnykProvider) copySnykConfigToContainer(containerID string, home string) error {
	configFile := fmt.Sprintf("%s/.config/configstore/snyk.json", home)
	if _, err := os.Stat(configFile); err == nil {
		options := types.CopyToContainerOptions{
			AllowOverwriteDirWithFile: false,
			CopyUIDGID:                true,
		}
		content, err := archive.Tar(configFile, archive.Gzip)
		if err != nil {
			return err
		}
		return d.cli.Client().CopyToContainer(d.context, containerID, "/root/.config/configstore/", content, options)
	}
	return nil

}

func (d *dockerSnykProvider) startContainer(containerID string) (func(), error) {
	resp, err := d.cli.Client().ContainerAttach(d.context, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			_, err = stdcopy.StdCopy(d.out, d.err, resp.Reader)
		}
	}()

	return resp.Close, d.cli.Client().ContainerStart(d.context, containerID, types.ContainerStartOptions{})
}

func (d *dockerSnykProvider) copySnykConfigToHost(containerID string, home string) error {
	reader, _, err := d.cli.Client().CopyFromContainer(d.context, containerID, "/root/.config/configstore/snyk.json")
	if err != nil {
		return err
	}
	configstoreFolder := filepath.Join(home, ".config", "configstore")
	err = os.MkdirAll(configstoreFolder, 0744)
	if err != nil {
		return err
	}

	// need NoLChown option to let tests pass when run as root, see https://github.com/habitat-sh/builder/issues/365#issuecomment-382862233
	return archive.Untar(reader, configstoreFolder, &archive.TarOptions{NoLchown: true})
}

func (d *dockerSnykProvider) Scan(image string) error {
	var token string
	snykAuthToken, err := getSnykAuthenticationToken()
	if snykAuthToken == "" || err != nil {
		var err error
		token, err = getToken(d.Options)
		if err != nil {
			return fmt.Errorf("failed to get DockerScanID: %s", err)
		}
		token = fmt.Sprintf("SNYK_DOCKER_TOKEN=%s", token)
	} else {
		token = fmt.Sprintf("SNYK_TOKEN=%s", snykAuthToken)
	}
	// check snyk token
	containerID, removeContainer, err := d.newCommand([]string{token}, append(d.flags, image)...)
	if err != nil {
		return err
	}
	//nolint: errcheck
	defer removeContainer()
	streamFunc, err := d.startContainer(containerID)
	if err != nil {
		return err
	}
	defer streamFunc()

	return d.checkContainerState(containerID)
}

func (d *dockerSnykProvider) Version() (string, error) {
	containerID, removeContainer, err := d.newCommand([]string{}, "--version")
	if err != nil {
		return "", err
	}
	//nolint: errcheck
	defer removeContainer()
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	d.out = buff
	d.err = buffErr
	streamFunc, err := d.startContainer(containerID)
	if err != nil {
		return "", err
	}
	defer streamFunc()

	err = d.checkContainerState(containerID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}

func (d *dockerSnykProvider) newCommand(envVars []string, arg ...string) (string, removeContainerFunc, error) {
	bindings := dockerBindings{
		"/var/run/docker.sock:/var/run/docker.sock",
	}
	for index, argument := range arg {
		if strings.HasPrefix(argument, "--file") {
			argSplit := strings.Split(argument, "=")
			filePath, err := filepath.Abs(argSplit[1])
			if err != nil {
				return "", nil, err
			}

			bindings = append(bindings, fmt.Sprintf(`%s:/app/Dockerfile`, filePath))
			arg[index] = "--file=/app/Dockerfile"
		}
	}
	defaultEnvs := []string{"NO_UPDATE_NOTIFIER=true", "SNYK_CFG_DISABLESUGGESTIONS=true",
		"SNYK_INTEGRATION_NAME=DOCKER_DESKTOP"}
	envVars = append(envVars, defaultEnvs...)

	args := strslice.StrSlice{"snyk"}
	args = append(args, arg...)
	config, hostConfig := containerConfigs(envVars, bindings, args)

	result, err := d.cli.Client().ContainerCreate(d.context, &config, &hostConfig, nil, &v1.Platform{Architecture: "amd64", OS: "linux"}, "")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create container: %s", err)
	}
	removeContainer := d.removeContainer(result.ID)
	return result.ID, removeContainer, nil
}

func containerConfigs(envVars dockerEnvs, bindings dockerBindings, entrypoint strslice.StrSlice) (container.Config, container.HostConfig) {
	config := container.Config{
		Image:        image,
		Env:          envVars,
		Entrypoint:   entrypoint,
		AttachStderr: true,
		AttachStdout: true,
	}

	hostConfig := container.HostConfig{
		Binds: bindings,
	}
	return config, hostConfig
}

func getSnykAuthenticationToken() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	snykConfFilePath := filepath.Join(home, ".config", "configstore", "snyk.json")
	buff, err := ioutil.ReadFile(snykConfFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var config snykConfig
	if err := json.Unmarshal(buff, &config); err != nil {
		return "", err
	}
	return config.API, nil
}
