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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/docker/scan-cli-plugin/internal/authentication"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/scan-cli-plugin/internal/hub"
)

// Provider abstracts a scan provider
type Provider interface {
	Authenticate(token string) error
	Scan(image string) error
	Version() (string, error)
}

// Options default options for all provider types
type Options struct {
	flags   []string
	auth    types.AuthConfig
	context context.Context
	out     io.Writer
	err     io.Writer
	path    string
}

// NewProvider returns default provider options setup with the give options
func NewProvider(options ...Ops) (Options, error) {
	provider := Options{
		flags: []string{"container", "test"},
		out:   os.Stdout,
		err:   os.Stderr,
	}
	for _, op := range options {
		if err := op(&provider); err != nil {
			return Options{}, err
		}
	}
	return provider, nil
}

// UseExternalBinary return true if the provider path option is setup
func UseExternalBinary(providerOpts Options) bool {
	return providerOpts.path != ""
}

// Ops defines options to setup a provider
type Ops func(provider *Options) error

// WithAuthConfig update the Snyk provider with the auth configuration from Docker CLI
func WithAuthConfig(authResolver func(*registry.IndexInfo) types.AuthConfig) Ops {
	return func(provider *Options) error {
		provider.auth = authResolver(hub.GetInstance().RegistryInfo)
		return nil
	}
}

//WithContext update the provider with a cancelable context
func WithContext(ctx context.Context) Ops {
	return func(options *Options) error {
		options.context = ctx
		return nil
	}
}

//WithStreams sets the out and err streams to be used by commands
func WithStreams(out, err io.Writer) Ops {
	return func(options *Options) error {
		options.out = out
		options.err = err
		return nil
	}
}

// WithJSON set JSONFormat to display scan result in JSON
func WithJSON() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--json")
		return nil
	}
}

// WithoutBaseImageVulnerabilities don't display the vulnerabilities from the base image
func WithoutBaseImageVulnerabilities() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--exclude-base-image-vulns")
		return nil
	}
}

// WithDockerFile improve result by providing a Dockerfile
func WithDockerFile(path string) Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--file="+path)
		return nil
	}
}

// WithDependencyTree shows the dependency tree before scan results
func WithDependencyTree() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--print-deps")
		return nil
	}
}

// WithFailOn only fail when there are vulnerabilities that can be fixed
func WithFailOn(failOn string) Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--fail-on="+failOn)
		return nil
	}
}

// WithSeverity only reports vulnerabilities of the provided level or higher
func WithSeverity(severity string) Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--severity-threshold="+severity)
		return nil
	}
}

// WithGroupIssues groups same issues in a single one when using --json flag
func WithGroupIssues() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--group-issues")
		return nil
	}
}

// WithAppVulns scans for applications vulnerabilities as well
func WithAppVulns() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--app-vulns")
		// We started with a default depth value of 2
		provider.flags = append(provider.flags, fmt.Sprintf("--nested-jars-depth=%d", 2))
		return nil
	}
}

// WithPath update the provider binary with the path from the configuration
func WithPath(path string) Ops {
	return func(provider *Options) error {
		if p, err := exec.LookPath("snyk"); err == nil && checkUserSnykBinaryVersion(p) {
			path = p
		}
		provider.path = path
		return nil
	}
}

// WithExperimental allows running `--json` flag in combination of `--app-vulns`
func WithExperimental() Ops {
	return func(provider *Options) error {
		provider.flags = append(provider.flags, "--experimental")
		return nil
	}
}

func getToken(opts Options) (string, error) {
	if opts.auth.Username == "" {
		return "", fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
	}
	h := hub.GetInstance()
	jwks, err := h.FetchJwks()
	if err != nil {
		return "", err
	}
	authenticator := authentication.NewAuthenticator(jwks, h.APIHubBaseURL)
	return authenticator.GetToken(opts.auth)
}
