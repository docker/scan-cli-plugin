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

	"github.com/docker/scan-cli-plugin/internal/authentication"
	"github.com/docker/scan-cli-plugin/internal/hub"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

type dockerSnykProvider struct {
	Options
}

// NewDockerSnykProvider returns a containerized Snyk implementation of scan provider
func NewDockerSnykProvider(ops ...Ops) (Provider, error) {
	provider := dockerSnykProvider{
		Options: Options{
			flags: []string{"container", "test"},
			out:   os.Stdout,
			err:   os.Stderr,
		},
	}
	for _, op := range ops {
		if err := op(&provider.Options); err != nil {
			return nil, err
		}
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
	/*cmd := s.newCommand("auth", token)
	cmd.Env = append(cmd.Env,
		"SNYK_UTM_MEDIUM=Partner",
		"SNYK_UTM_SOURCE=Docker",
		"SNYK_UTM_CAMPAIGN=Docker-Desktop-2020")
	cmd.Stdout = s.out
	cmd.Stderr = s.err
	return checkCommandErr(cmd.Run())*/
	return nil
}

func (d *dockerSnykProvider) Scan(image string) error {
	var token string
	snykAuthToken, err := getSnykAuthenticationToken()
	if snykAuthToken == "" || err != nil {
		var err error
		token, err = d.getDockerToken()
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
	args := []string{"run", "-i", "--rm", "-e", "NO_UPDATE_NOTIFIER=true", "-e", "SNYK_CFG_DISABLESUGGESTIONS=true",
		"-e", "SNYK_INTEGRATION_NAME=DOCKER_DESKTOP",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
	}
	for _, envVar := range envVars {
		args = append(args, "-e", envVar)
	}

	args = append(args, "snyk/snyk:alpine")
	args = append(args, "snyk")
	args = append(args, arg...)
	cmd := exec.CommandContext(d.context, "docker", args...)
	return cmd
}

func (d *dockerSnykProvider) getDockerToken() (string, error) {
	if d.auth.Username == "" {
		return "", fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
	}
	h := hub.GetInstance()
	jwks, err := h.FetchJwks()
	if err != nil {
		return "", err
	}
	authenticator := authentication.NewAuthenticator(jwks, h.APIHubBaseURL)
	return authenticator.GetToken(d.auth)
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
