package hub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docker/docker/api/types/registry"
	"gopkg.in/square/go-jose.v2"
)

//Instance stores all the specific pieces needed to dialog with Hub
type Instance struct {
	APIHubBaseURL string
	JwksURL       string
	RegistryInfo  *registry.IndexInfo
}

//GetInstance returns the current hub instance, which can be overridden by
// DOCKER_SCAN_HUB_INSTANCE env var
func GetInstance() *Instance {
	override := os.Getenv("DOCKER_SCAN_HUB_INSTANCE")
	switch override {
	case "staging":
		return &staging
	case "prod":
		return &prod
	default:
		return &prod
	}
}

//FetchJwks fetches a jwks.json file and parses it
func (i *Instance) FetchJwks() (jose.JSONWebKeySet, error) {
	// fetch jwks.json file from URL
	resp, err := http.Get(i.JwksURL)
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("failed to fetch JWKS: %s", err)
	}
	if resp.StatusCode < http.StatusOK && resp.StatusCode >= 300 {
		return jose.JSONWebKeySet{}, fmt.Errorf("failed to fetch JWKS: invalid status code %v", resp.StatusCode)
	}
	if resp.Body == nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("failed to fetch JWKS: invalid jwks.json file")
	}
	defer resp.Body.Close() //nolint: errcheck

	// Read and parse jwks.json file
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("failed to read JWKS: %s", err)
	}
	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal(buf, &keySet); err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("invalid JWKS: %s", err)
	}
	return keySet, nil
}

var (
	staging = Instance{
		APIHubBaseURL: "https://hub-stage.docker.com",
		RegistryInfo: &registry.IndexInfo{
			Name:     "index-stage.docker.io",
			Mirrors:  nil,
			Secure:   true,
			Official: false,
		},
		JwksURL: "https://jwt-stage.docker.com/scan/.well-known/jwks.json",
	}

	prod = Instance{
		APIHubBaseURL: "https://hub.docker.com",
		RegistryInfo: &registry.IndexInfo{
			Name:     "index.docker.io",
			Mirrors:  nil,
			Secure:   true,
			Official: true,
		},
		JwksURL: "https://jwt.docker.com/scan/.well-known/jwks.json",
	}
)
