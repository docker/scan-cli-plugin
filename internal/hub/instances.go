package hub

import (
	"os"

	"github.com/docker/docker/api/types/registry"
)

//Instance stores all the specific pieces needed to dialog with Hub
type Instance struct {
	APIHubBaseURL string
	RegistryInfo  *registry.IndexInfo
	JWKS          string
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

var (
	staging = Instance{
		APIHubBaseURL: "https://hub-stage.docker.com",
		RegistryInfo: &registry.IndexInfo{
			Name:     "index-stage.docker.io",
			Mirrors:  nil,
			Secure:   true,
			Official: false,
		},
		JWKS: `{
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
}`,
	}

	prod = Instance{
		APIHubBaseURL: "https://hub.docker.com",
		RegistryInfo: &registry.IndexInfo{
			Name:     "index.docker.io",
			Mirrors:  nil,
			Secure:   true,
			Official: true,
		},
		JWKS: `{
 "keys": [
   {
     "use": "sig",
     "kty": "EC",
     "kid": "/Il5tHgzaqqjh6vp1Je9pG0Ic+s/eRQ7C1dLkmITuop0z8qLNszOuqIJldWSEPitEN/cCW5BKt0buUoVHy9o6A==",
     "crv": "P-256",
     "alg": "ES256",
     "x": "oWouB0UC--Gg7hhYiOKExx2dXVsSdP4t7xfIYbVVXSI",
     "y": "b7WeNOKN2Ur00AFO-8-1o_hdflRCz9gtq-JE-3dFvRU"
   }
 ]
}`,
	}
)
