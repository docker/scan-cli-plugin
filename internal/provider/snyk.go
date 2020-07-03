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

	"github.com/docker/docker-scan/internal/authentication"
	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

type snykProvider struct {
	path  string
	flags []string
	auth  types.AuthConfig
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

// WithAuthConfig update the Snyk provider with the auth configuration from Docker CLI
func WithAuthConfig(authConfig types.AuthConfig) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.auth = authConfig
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
	cmd := exec.Command(s.path, "auth", token)
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

	cmd := exec.Command(s.path, append(s.flags, image)...)
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
	authenticator := authentication.NewAuthenticator(jwksStaging)
	return authenticator.GetToken(s.auth)
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

const (
	jwksStaging = `{
  "keys": [
    {
      "use": "sig",
      "kty": "EC",
      "kid": "yy49bsZVoCPg6PgH1iXtuBlOAMVPsMpNb78iUvqrTn/3iDmS6N5nPVjtpcZqgXyAUl4S6tbihdSSPk3nTsGOxA==",
      "crv": "P-256",
      "alg": "ES256",
      "x": "NjptJx3r6yRl895HksB9pK6UmxGZgRMznkRzQCAnHbg",
      "y": "RuuhcGfpxiNZ8__hGRkzc-TGxMVOVWThNEj1-tL_Sk0"
    }
  ]
}`

	//	jwksProd = `{
	//  "keys": [
	//    {
	//      "use": "sig",
	//      "kty": "EC",
	//      "kid": "/Il5tHgzaqqjh6vp1Je9pG0Ic+s/eRQ7C1dLkmITuop0z8qLNszOuqIJldWSEPitEN/cCW5BKt0buUoVHy9o6A==",
	//      "crv": "P-256",
	//      "alg": "ES256",
	//      "x": "oWouB0UC--Gg7hhYiOKExx2dXVsSdP4t7xfIYbVVXSI",
	//      "y": "b7WeNOKN2Ur00AFO-8-1o_hdflRCz9gtq-JE-3dFvRU"
	//    }
	//  ]
	//}`
)
