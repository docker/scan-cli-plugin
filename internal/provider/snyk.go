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
	"github.com/mitchellh/go-homedir"
)

const dockerHubAuthURL = "https://index.docker.io/v1/"

type snykProvider struct {
	path  string
	auths map[string]types.AuthConfig
}

type snykConfig struct {
	API string `json:"api,omitempty"`
}

// NewSnykProvider returns a Snyk implementation of scan provider
func NewSnykProvider(path string, authsConfig map[string]types.AuthConfig) Provider {
	if p, err := exec.LookPath("snyk"); err == nil {
		path = p
	}
	return &snykProvider{path, authsConfig}
}

func (s *snykProvider) Version() (string, error) {
	cmd := exec.Command(s.path, "--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		if err == exec.ErrNotFound {
			// Could not find Snyk in $PATH
			return "", fmt.Errorf("could not find Snyk binary")
		} else if _, ok := err.(*os.PathError); ok {
			// The specified path for Snyk binary does not exist
			return "", fmt.Errorf("could not find Snyk binary")
		}
		return "", fmt.Errorf("fail to call Snyk: %s %s", err, buffErr.String())
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}

func (s *snykProvider) Scan(image string) error {
	if ok, err := s.isAuthenticated(s.auths); !ok || err != nil {
		if err != nil {
			return err
		}
		return &authenticationError{}
	}
	return nil
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
