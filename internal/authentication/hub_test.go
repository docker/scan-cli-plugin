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
	"gotest.tools/v3/fs"
)

/* Tests:

- sauver dans un fichier scan-id.json
- verifier la signature du content + vérifier l'expiration, sinon relogger
- verifier si loggé dans snyk?
*/

func TestHubAuthenticateNegociatesToken(t *testing.T) {
	authConfig := types.AuthConfig{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/users/login":
			assert.Equal(t, r.Method, http.MethodPost)
			buf, err := ioutil.ReadAll(r.Body)
			assert.NilError(t, err)
			var actualAuthConfig types.AuthConfig
			assert.NilError(t, json.Unmarshal(buf, &actualAuthConfig))
			assert.DeepEqual(t, actualAuthConfig, authConfig)
			fmt.Fprint(w, `{"content":"hub-content"}`)

		case "/v2/scan/provider/content":
			assert.Equal(t, r.Method, http.MethodGet)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer hub-content")
			fmt.Fprint(w, `XXXX.YYYY.ZZZZ`)

		default:
			t.FailNow()
		}
	}))
	defer ts.Close()

	authenticator := NewAuthenticator()
	authenticator.hub.domain = ts.URL
	token, err := authenticator.negotiateScanIdToken(authConfig)
	assert.NilError(t, err)
	assert.Equal(t, token, "XXXX.YYYY.ZZZZ")
}

func TestHubAuthenticateChecksTokenValidity(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "missing file",
			content:  "",
			expected: "",
		},
		{
			name:     "invalid content",
			content:  "invalid content",
			expected: "",
		},
		{
			name:     "valid content with unknown user",
			content:  `{"hubUser1": "ZZZZ.YYYY.XXXX"}`,
			expected: "",
		},
		{
			name: "valid content with hub user",
			content: `{
	"hubUser1": "ZZZZ.YYYY.XXXX",
	"hubUser2": "XXXX.YYYY.ZZZZ"
}`,
			expected: "XXXX.YYYY.ZZZZ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var dir *fs.Dir
			if testCase.content != "" {
				dir = fs.NewDir(t, testCase.name, fs.WithFile("tokens.json", testCase.content))
			} else {
				dir = fs.NewDir(t, testCase.name)
			}
			defer dir.Remove()

			authenticator := NewAuthenticator()
			authenticator.tokensPath = dir.Join("tokens.json")

			authConfig := types.AuthConfig{Username: "hubUser2"}

			token, err := authenticator.getLocalToken(authConfig)
			assert.NilError(t, err)
			assert.Equal(t, token, testCase.expected)
		})
	}
}

func TestUpdateLocalToken(t *testing.T){
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "no file",
			content:  "",
			expected: `{"hubUser1":"ZZZZ.YYYY.XXXX"}`,
		},
		{
			name:     "invalid content",
			content:  "invalid content",
			expected: `{"hubUser1":"ZZZZ.YYYY.XXXX"}`,
		},
		{
			name:     "update content with new user",
			content:  `{"hubUser2":"XXXX.YYYY.ZZZZ"}`,
			expected: `{"hubUser1":"ZZZZ.YYYY.XXXX","hubUser2":"XXXX.YYYY.ZZZZ"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var dir *fs.Dir
			if testCase.content != "" {
				dir = fs.NewDir(t, testCase.name, fs.WithFile("tokens.json", testCase.content))
			} else {
				dir = fs.NewDir(t, testCase.name)
			}
			defer dir.Remove()

			authenticator := NewAuthenticator()
			authenticator.tokensPath = dir.Join("tokens.json")

			authConfig := types.AuthConfig{Username: "hubUser1"}

			err := authenticator.updateLocalToken(authConfig, "ZZZZ.YYYY.XXXX")
			assert.NilError(t, err)
			actual, err := ioutil.ReadFile(dir.Join("tokens.json"))
			assert.NilError(t, err)
			assert.Equal(t, string(actual), testCase.expected)
		})
	}
}