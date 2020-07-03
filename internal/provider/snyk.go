package provider

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
)

type snykProvider struct {
	path        string
	flags       []string
	scanIDToken string
}

type snykConfig struct {
	API string `json:"api,omitempty"`
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

func WithDockerScanIDToken(token string) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.scanIDToken = token
		return nil
	}
}

func (s *snykProvider) Authenticate(token string) error {
	if token != "" {
		if _, err := uuid.Parse(token); err != nil {
			return &invalidTokenError{token}
		}
	}
	cmd := exec.Command(s.path, "auth", token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return checkCommandErr(cmd.Run())
}

func (s *snykProvider) Scan(image string) error {
	cmd := exec.Command(s.path, append(s.flags, image)...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("SNYK_DOCKER_TOKEN=%s", s.scanIDToken))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return checkCommandErr(cmd.Run())
}

func (s *snykProvider) Version() (string, error) {
	cmd := exec.Command(s.path, "--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		return "", checkCommandErr(err)
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
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
