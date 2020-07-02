package authentication

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	cliConfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
)

const (
	apiHubBaseUrl = "https://hub.docker.com"
)

//Authenticator logs on docker Hub and retrieves a DockerScanID
// if the one stored locally has expired
type Authenticator struct {
	hub        hubClient
	tokensPath string
}

//NewAuthenticator returns an Authenticator
// configured to run against Docker Hub prod or staging
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		hub:        hubClient{},
		tokensPath: filepath.Join(cliConfig.Dir(), "scan", "tokens.json"),
	}
}

//Authenticate checks the local DockerScanID content for expiry,
// if expired it negotiates a new one on Docker Hub.
func (a *Authenticator) Authenticate(hubAuthConfig types.AuthConfig) (string, error) {
	// Retrieve token from local storage
	token, err := a.getLocalToken(hubAuthConfig)
	if err != nil {
		return "", nil
	}
	// TODO: check validity and expiration
	if token != "" {
		return token, nil
	}
	// Fetch a new token from Hub
	token, err = a.negotiateScanIdToken(hubAuthConfig)
	if err != nil {
		return "", nil
	}
	// Persist token on local storage
	if err := a.updateLocalToken(hubAuthConfig, token); err != nil {
		return "", err
	}
	return token, nil
}

func (a *Authenticator) getLocalToken(hubAuthConfig types.AuthConfig) (string, error) {
	buf, err := ioutil.ReadFile(a.tokensPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	tokens := map[string]string{}
	if err := json.Unmarshal(buf, &tokens); err != nil {
		return "", nil
	}
	return tokens[hubAuthConfig.Username], nil
}

func (a *Authenticator) negotiateScanIdToken(hubAuthConfig types.AuthConfig) (string, error) {
	hubToken, err := a.hub.login(hubAuthConfig)
	if err != nil {
		return "", err
	}
	return a.hub.getScanID(hubToken)
}

func (a *Authenticator) updateLocalToken(hubAuthConfig types.AuthConfig, token string) error {
	stats, err := os.Stat(a.tokensPath)
	mode := os.FileMode(0644)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		mode = stats.Mode()
	}

	buf, err := ioutil.ReadFile(a.tokensPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tokens := map[string]string{}
	_ = json.Unmarshal(buf, &tokens) // if an error occurs (invalid content), we just erase the content with a new map
	tokens[hubAuthConfig.Username] = token
	buf, err = json.Marshal(tokens)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(a.tokensPath, buf, mode)
}
