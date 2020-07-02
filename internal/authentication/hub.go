package authentication

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	cliConfig "github.com/docker/cli/cli/config"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
)

const (
	apiHubBaseUrl = "https://hub.docker.com"
)

//Authenticator logs on docker Hub and retrieves a DockerScanID
// if the one stored locally has expired
type Authenticator struct {
	hub hubClient
	tokensPath string
}

//NewAuthenticator returns an Authenticator
// configured to run against Docker Hub prod or staging
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		hub: hubClient{},
		tokensPath: filepath.Join(cliConfig.Dir(), "scan", "tokens.json"),
	}
}

//Authenticate checks the local DockerScanID content for expiry,
// if expired it negociates a new one on Docker Hub.
func (a *Authenticator) Authenticate(hubAuthConfig types.AuthConfig) (string, error) {
	// Retrieve token from local storage
	token, err := a.getLocalToken(hubAuthConfig)
	if err != nil{
		return "", nil
	}
	// TODO: check validity and expiration
	if token != ""{
		return token, nil
	}
	// Fetch a new token from Hub
	token, err = a.negociateScanIdToken(hubAuthConfig)
	if err != nil{
		return "", nil
	}
	// Persist token on local storage
	if err := a.updateLocalToken(hubAuthConfig, token); err != nil{
		return "", err
	}
	return token, nil
}

func (a *Authenticator) getLocalToken(hubAuthConfig types.AuthConfig) (string, error) {
	buf, err := ioutil.ReadFile(a.tokensPath)
	if errors.Is(err, os.ErrNotExist){
		return "", nil
	}
	tokens := map[string]string{}
	if err := json.Unmarshal(buf, &tokens); err != nil{
		return "", nil
	}
	return tokens[hubAuthConfig.Username], nil
}

func (a *Authenticator) negociateScanIdToken(hubAuthConfig types.AuthConfig) (string, error) {
	hubToken, err := a.hub.login(hubAuthConfig)
	if err != nil {
		return "", err
	}
	return a.hub.getScanID(hubToken)
}

func (a *Authenticator) updateLocalToken(hubAuthConfig types.AuthConfig, token string) error{
	stats, err := os.Stat(a.tokensPath)
	mode := os.FileMode(0644)
	if err != nil{
		if !errors.Is(err, os.ErrNotExist){
			return err
		}
	}else {
		mode = stats.Mode()
	}

	buf, err := ioutil.ReadFile(a.tokensPath)
	if err != nil && !errors.Is(err, os.ErrNotExist){
		return err
	}
	tokens := map[string]string{}
	_ = json.Unmarshal(buf, &tokens) // if an error occurs (invalid content), we just erase the content with a new map
	tokens[hubAuthConfig.Username] = token
	buf, err = json.Marshal(tokens)
	if err != nil{
		return err
	}
	return ioutil.WriteFile(a.tokensPath, buf, mode)
}

type hubClient struct {
	domain string
}

func (h *hubClient) login(hubAuthConfig types.AuthConfig) (string, error) {
	data, err := json.Marshal(hubAuthConfig)
	if err != nil {
		return "", err
	}
	body := bytes.NewBuffer(data)

	// Login on the Docker Hub
	req, err := http.NewRequest("POST", h.domain+"/v2/users/login", ioutil.NopCloser(body))
	if err != nil {
		return "", err
	}
	req.Header["Content-Type"] = []string{"application/json"}
	buf, err := doRequest(req)
	if err != nil {
		return "", err
	}

	creds := struct {
		Token string `json:"content"`
	}{}
	if err := json.Unmarshal(buf, &creds); err != nil {
		return "", err
	}
	return creds.Token, nil
}

func (h *hubClient) getScanID(hubToken string) (string, error) {
	req, err := http.NewRequest("GET", h.domain+"/v2/scan/provider/content", nil)
	if err != nil {
		return "", err
	}
	req.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", hubToken)}
	token, err := doRequest(req)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func doRequest(req *http.Request) ([]byte, error) {
	req.Header["Accept"] = []string{"application/json"}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code %q", resp.Status)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
