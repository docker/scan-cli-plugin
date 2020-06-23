package authentication

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
	"gotest.tools/v3/assert"
)

/* Tests:

- sauver dans un fichier scan-id.json
- verifier la signature du token + vérifier l'expiration, sinon relogger
- verifier si loggé dans snyk?
*/

func TestHubAuthenticateReturnsToken(t *testing.T) {
	authConfig := types.AuthConfig{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String(){
		case "/v2/users/login":
			assert.Equal(t, r.Method, http.MethodPost)
			buf, err := ioutil.ReadAll(r.Body)
			assert.NilError(t, err)
			var actualAuthConfig types.AuthConfig
			assert.NilError(t, json.Unmarshal(buf, &actualAuthConfig))
			assert.DeepEqual(t, actualAuthConfig, authConfig)
			fmt.Fprint(w, `{"token":"hub-token"}`)

		case "/v2/scan/provider/token":
			assert.Equal(t, r.Method, http.MethodGet)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer hub-token")
			fmt.Fprint(w, `XXXX.YYYY.ZZZZ`)

		default:
			t.FailNow()
		}
	}))
	defer ts.Close()

	authenticator := NewAuthenticator()
	authenticator.hub.domain = ts.URL
	token, err := authenticator.Authenticate(authConfig)
	assert.NilError(t, err)
	assert.Equal(t, token, "XXXX.YYYY.ZZZZ")
}
