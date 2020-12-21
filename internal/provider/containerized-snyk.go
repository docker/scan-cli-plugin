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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

// ImageDigest is the sha snyk/snyk:alpine image, set at build time
var ImageDigest = "unknown"

type dockerSnykProvider struct {
	Options
}

type dockerCmd struct {
	cmdDocker string
	cmd       string
	image     string
	flags     dockerFlags
	envs      dockerEnvs
	args      dockerArgs
}

type dockerFlags []string
type dockerEnvs []string
type dockerArgs []string

func newDockerCmd(cmdDocker string, cmd string, flags dockerFlags, envs dockerEnvs, args ...string) dockerCmd {
	return dockerCmd{
		cmdDocker: cmdDocker,
		cmd:       cmd,
		image:     fmt.Sprintf("snyk/snyk@%s", ImageDigest),
		flags:     flags,
		envs:      envs,
		args:      args,
	}
}

func (d dockerCmd) toShellCmd() []string {
	arguments := []string{d.cmdDocker}
	arguments = append(arguments, d.flags...)
	for _, env := range d.envs {
		arguments = append(arguments, "-e", env)
	}
	if d.cmdDocker != "start" {
		arguments = append(arguments, d.image, d.cmd)
	}
	arguments = append(arguments, d.args...)
	return arguments
}

// NewDockerSnykProvider returns a containerized Snyk implementation of scan provider
func NewDockerSnykProvider(defaultProvider Options) (Provider, error) {
	provider := dockerSnykProvider{
		Options: defaultProvider,
	}
	return &provider, nil
}

//
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

	err = d.createContainer(token, containerName)
	if err != nil {
		return err
	}

	err = d.copySnykConfigToContainer(containerName, home)
	if err != nil {
		return d.deleteContainer(containerName, err)
	}

	err = d.startAuthenticateContainer(containerName)
	if err != nil {
		return d.deleteContainer(containerName, err)
	}
	return d.copySnykConfigToHost(containerName, home)
}

func (d *dockerSnykProvider) createContainer(token string, containerName string) error {
	envVars := dockerEnvs{
		"NO_UPDATE_NOTIFIER=true",
		"SNYK_CFG_DISABLESUGGESTIONS=true",
		"SNYK_INTEGRATION_NAME=DOCKER_DESKTOP",
		"SNYK_UTM_MEDIUM=Partner",
		"SNYK_UTM_SOURCE=Docker",
		"SNYK_UTM_CAMPAIGN=Docker-Desktop-2020",
	}
	flags := dockerFlags{"-i", "--name", containerName,
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-v", "TMP:/root/.config/configstore",
	}
	cmdDocker := newDockerCmd("create", "snyk", flags, envVars, "auth", token)
	cmd := exec.CommandContext(d.context, "docker", cmdDocker.toShellCmd()...)
	return cmd.Run()
}

func (d *dockerSnykProvider) copySnykConfigToContainer(containerName string, home string) error {
	configFile := fmt.Sprintf("%s/.config/configstore/snyk.json", home)
	if _, err := os.Stat(configFile); err == nil {
		cpArgs := []string{"cp", "-a", configFile,
			fmt.Sprintf("%s:/root/.config/configstore/snyk.json", containerName),
		}
		cmd := exec.CommandContext(d.context, "docker", cpArgs...)
		return cmd.Run()
	}
	return nil
}

func (d *dockerSnykProvider) startAuthenticateContainer(containerName string) error {
	flags := dockerFlags{"-a", containerName}
	cmdDocker := newDockerCmd("start", "", flags, dockerEnvs{})
	cmd := exec.CommandContext(d.context, "docker", cmdDocker.toShellCmd()...)
	cmd.Stdout = d.out
	cmd.Stderr = d.err
	return checkCommandErr(cmd.Run())
}

func (d *dockerSnykProvider) copySnykConfigToHost(containerName string, home string) error {
	cpArgs := []string{"cp", fmt.Sprintf("%s:/root/.config/configstore/snyk.json", containerName),
		fmt.Sprintf("%s/.config/configstore/snyk.json", home)}
	cmd := exec.CommandContext(d.context, "docker", cpArgs...)
	if err := cmd.Run(); err != nil {
		return d.deleteContainer(containerName, err)
	}
	return d.deleteContainer(containerName, nil)
}

func (d *dockerSnykProvider) deleteContainer(containerName string, initialError error) error {
	cmd := exec.CommandContext(d.context, "docker", "rm", containerName)
	err := cmd.Run()
	if initialError != nil {
		return initialError
	}
	return err
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
	cmd := d.newCommand([]string{token}, append(d.flags, image)...)

	cmd.Stdout = d.out
	cmd.Stderr = d.err
	return checkCommandErr(cmd.Run())
}

func (d *dockerSnykProvider) Version() (string, error) {
	cmd := d.newCommand([]string{}, "--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("failed to get snyk version: %s", checkCommandErr(err))
		if buffErr.String() != "" {
			errMsg = fmt.Sprintf(errMsg+",%s", buffErr.String())
		}
		return "", fmt.Errorf(errMsg)
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}

func (d *dockerSnykProvider) newCommand(envVars []string, arg ...string) *exec.Cmd {
	flags := []string{"-i", "--rm", "-v", "/var/run/docker.sock:/var/run/docker.sock"}
	defaultEnvs := []string{"NO_UPDATE_NOTIFIER=true", "SNYK_CFG_DISABLESUGGESTIONS=true",
		"SNYK_INTEGRATION_NAME=DOCKER_DESKTOP"}
	envVars = append(envVars, defaultEnvs...)

	cmdDocker := newDockerCmd("run", "snyk", flags,
		envVars, arg...)
	cmd := exec.CommandContext(d.context, "docker", cmdDocker.toShellCmd()...)
	return cmd
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
