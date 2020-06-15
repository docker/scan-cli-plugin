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

	"github.com/docker/cli/cli/config/types"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

const dockerHubAuthURL = "https://index.docker.io/v1/"

type snykProvider struct {
	path       string
	auths      map[string]types.AuthConfig
	JSONFormat bool
}

type snykConfig struct {
	API string `json:"api,omitempty"`
}

// NewSnykProvider returns a Snyk implementation of scan provider
//path string, authsConfig map[string]types.AuthConfig
func NewSnykProvider(ops ...SnykProviderOps) (Provider, error) {
	var provider snykProvider
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

// WithAuthConfig update the Snyk provider with the auths configuration from Docker CLI
func WithAuthConfig(authsConfig map[string]types.AuthConfig) SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.auths = authsConfig
		return nil
	}
}

// WithJSON set JSONFormat to display scan result in JSON
func WithJSON() SnykProviderOps {
	return func(provider *snykProvider) error {
		provider.JSONFormat = true
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
	if ok, err := s.isAuthenticated(s.auths); !ok || err != nil {
		if err != nil {
			return err
		}
		return &authenticationError{}
	}
	flags := []string{"test", "--docker"}
	if s.JSONFormat {
		flags = append(flags, "--json")
	}
	flags = append(flags, image)
	cmd := exec.Command(s.path, flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return checkCommandErr(cmd.Run())
}

func (s *snykProvider) isAuthenticated(auths map[string]types.AuthConfig) (bool, error) {
	if ok := isAuthenticatedOnHub(auths); ok {
		return true, nil
	}
	return isAuthenticatedOnSnyk()
}

func isAuthenticatedOnHub(auths map[string]types.AuthConfig) bool {
	_, ok := auths[dockerHubAuthURL]
	return ok
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
