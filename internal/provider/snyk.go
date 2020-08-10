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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/scan-cli-plugin/internal/authentication"
	"github.com/docker/scan-cli-plugin/internal/hub"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

type snykProvider struct {
	path    string
	flags   []string
	auth    types.AuthConfig
	context context.Context
}

// NewSnykProvider returns a Snyk implementation of scan provider
func NewSnykProvider(ops ...SnykProviderOps) (Provider, error) {
	provider := snykProvider{
		flags: []string{"test", "--docker"},
	}
	for _, op := range ops {
		if err := op(&provider); err != nil {
			return nil, err
		}
	}
	return &provider, nil
}

// SnykProviderOps function taking a pointer to a Snyk Provider and returning an error if needed
type SnykProviderOps func(*snykProvider) error

//WithContext update the Snyk provider with a cancelable context
func WithContext(ctx context.Context) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.context = ctx
		return nil
	}
}

// WithPath update the Snyk provider with the path from the configuration
func WithPath(path string) SnykProviderOps {
	return func(provider *snykProvider) error {
		if p, err := exec.LookPath("snyk"); err == nil {
			path = p
		}
		provider.path = path
		return nil
	}
}

// WithAuthConfig update the Snyk provider with the auth configuration from Docker CLI
func WithAuthConfig(authResolver func(*registry.IndexInfo) types.AuthConfig) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.auth = authResolver(hub.GetInstance().RegistryInfo)
		return nil
	}
}

// WithJSON set JSONFormat to display scan result in JSON
func WithJSON() SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.flags = append(provider.flags, "--json")
		return nil
	}
}

// WithoutBaseImageVulnerabilities don't display the vulnerabilities from the base image
func WithoutBaseImageVulnerabilities() SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.flags = append(provider.flags, "--exclude-base-image-vulns")
		return nil
	}
}

// WithDockerFile improve result by providing a Dockerfile
func WithDockerFile(path string) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.flags = append(provider.flags, "--file="+path)
		return nil
	}
}

// WithDependencyTree shows the dependency tree before scan results
func WithDependencyTree() SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.flags = append(provider.flags, "--print-deps")
		return nil
	}
}

func (s *snykProvider) Authenticate(token string) error {
	if token != "" {
		if _, err := uuid.Parse(token); err != nil {
			return &invalidTokenError{token}
		}
	}
	cmd := s.newCommand("auth", token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return checkCommandErr(cmd.Run())
}

func (s *snykProvider) Scan(image string) error {
	// check snyk token
	var token string
	if authenticated, err := isAuthenticatedOnSnyk(); !authenticated || err != nil {
		var err error
		token, err = s.getToken()
		if err != nil {
			return fmt.Errorf("failed to get DockerScanID: %s", err)
		}
	}
	cmd := s.newCommand(append(s.flags, image)...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("SNYK_DOCKER_TOKEN=%s", token))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return checkCommandErr(cmd.Run())
}

func (s *snykProvider) getToken() (string, error) {
	if s.auth.Username == "" {
		return "", fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
	}
	h := hub.GetInstance()
	jwks, err := h.FetchJwks()
	if err != nil {
		return "", err
	}
	authenticator := authentication.NewAuthenticator(jwks, h.APIHubBaseURL)
	return authenticator.GetToken(s.auth)
}

func (s *snykProvider) Version() (string, error) {
	cmd := s.newCommand("--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get snyk version: %s, %s", checkCommandErr(err), buffErr.String())
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}

func (s *snykProvider) newCommand(arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(s.context, s.path, arg...)
	cmd.Env = append(os.Environ(), "NO_UPDATE_NOTIFIER=1")
	return cmd
}

func checkCommandErr(err error) error {
	if err == nil {
		return nil
	}
	if err == exec.ErrNotFound {
		// Could not find Snyk in $PATH
		return fmt.Errorf("could not find Snyk binary")
	} else if _, ok := err.(*exec.Error); ok {
		return fmt.Errorf("could not find Snyk binary")
	} else if _, ok := err.(*os.PathError); ok {
		// The specified path for Snyk binary does not exist
		return fmt.Errorf("could not find Snyk binary")
	}
	return err
}

type snykConfig struct {
	API string `json:"api,omitempty"`
}

func isAuthenticatedOnSnyk() (bool, error) {
	home, err := homedir.Dir()
	if err != nil {
		return false, err
	}
	snykConfFilePath := filepath.Join(home, ".config", "configstore", "snyk.json")
	buff, err := ioutil.ReadFile(snykConfFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var config snykConfig
	if err := json.Unmarshal(buff, &config); err != nil {
		return false, err
	}

	return config.API != "", nil
}
