package authentication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/docker/api/types"
)

const (
	apiHubBaseUrl = "https://hub.docker.com"
)

//Authenticator logs on docker Hub and retrieves a DockerScanID
// if the one stored locally has expired
type Authenticator struct {
	hub hubClient
}

//NewAuthenticator returns an Authenticator
// configured to run against Docker Hub prod or staging
func NewAuthenticator() *Authenticator {
	return &Authenticator{hub: hubClient{}}
}

//Authenticate checks the local DockerScanID token for expiry,
// if expired it negociates a new one on Docker Hub.
func (a *Authenticator) Authenticate(hubAuthConfig types.AuthConfig) (string, error) {
	// TODO: check expiry
	hubToken, err := a.hub.login(hubAuthConfig)
	if err != nil {
		return "", err
	}
	return a.hub.getScanID(hubToken)
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
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(buf, &creds); err != nil {
		return "", err
	}
	return creds.Token, nil
}

func (h *hubClient) getScanID(hubToken string) (string, error) {
	req, err := http.NewRequest("GET", h.domain+"/v2/scan/provider/token", nil)
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
