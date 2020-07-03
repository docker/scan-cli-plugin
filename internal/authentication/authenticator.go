package authentication

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	cliConfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	apiHubBaseUrl = "https://hub.docker.com"
	expirationWindow = -1 * time.Minute
)

//Authenticator logs on docker Hub and retrieves a DockerScanID
// if the one stored locally has expired
type Authenticator struct {
	hub        hubClient
	tokensPath string
	jwks       string
}

//NewAuthenticator returns an Authenticator
// configured to run against Docker Hub prod or staging
func NewAuthenticator(jwks string) *Authenticator {
	return &Authenticator{
		hub:        hubClient{},
		tokensPath: filepath.Join(cliConfig.Dir(), "scan", "tokens.json"),
		jwks:       jwks,
	}
}

//GetToken checks the local DockerScanID content for expiry,
// if expired it negotiates a new one on Docker Hub.
func (a *Authenticator) GetToken(hubAuthConfig types.AuthConfig) (string, error) {
	// Retrieve token from local storage
	token, err := a.getLocalToken(hubAuthConfig)
	if err != nil {
		return "", nil
	}
	// Check if the token is well formed and still valid
	if err := a.checkTokenValidity(token); err == nil {
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

func (a *Authenticator) checkTokenValidity(token string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}

	parsedToken, err := jwt.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("invalid token: %s", err)
	}
	publicKey, err := a.findKey(parsedToken)
	if err != nil {
		return err
	}
	out := jwt.Claims{}
	if err := parsedToken.Claims(publicKey, &out); err != nil {
		return fmt.Errorf("invalid token: signature does not match the content: %s", err)
	}
	if err := out.ValidateWithLeeway(jwt.Expected{Time: time.Now()}, expirationWindow); err != nil {
		return fmt.Errorf("token has expired: %s", err)
	}
	return nil
}

func (a *Authenticator) findKey(token *jwt.JSONWebToken) (crypto.PublicKey, error) {
	jwks := jose.JSONWebKeySet{}
	if err := json.Unmarshal([]byte(a.jwks), &jwks); err != nil {
		return nil, err
	}
	var kid string
	for _, header := range token.Headers {
		if header.KeyID != "" {
			kid = header.KeyID
			break
		}
	}
	if kid == "" {
		return nil, fmt.Errorf("invalid token: key identifier does not match")
	}
	for _, key := range jwks.Keys {
		if key.KeyID == kid {
			return key.Public(), nil
		}
	}
	return nil, fmt.Errorf("invalid token: key identifier does not match")
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
