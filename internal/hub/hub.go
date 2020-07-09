package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/docker/api/types"
)

//Client sends authenticates on Hub and sends requests to the API
type Client struct {
	Domain string
}

//Login logs into Hub and returns the auth token
func (h *Client) Login(hubAuthConfig types.AuthConfig) (string, error) {
	data, err := json.Marshal(hubAuthConfig)
	if err != nil {
		return "", err
	}
	body := bytes.NewBuffer(data)

	// Login on the Docker Hub
	req, err := http.NewRequest("POST", h.Domain+"/v2/users/login", ioutil.NopCloser(body))
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

//GetScanID calls the scan service which returns a DockerScanID as a JWT token
func (h *Client) GetScanID(hubToken string) (string, error) {
	req, err := http.NewRequest("GET", h.Domain+"/v2/scan/provider/token", nil)
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
		defer resp.Body.Close() //nolint:errcheck
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
