/*
   Copyright 2020 Docker Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package authentication

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/scan-cli-plugin/internal/hub"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestHubAuthenticateNegociatesToken(t *testing.T) {
	authConfig := types.AuthConfig{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case hub.LoginURL:
			assert.Equal(t, r.Method, http.MethodPost)
			buf, err := ioutil.ReadAll(r.Body)
			assert.NilError(t, err)
			var actualAuthConfig types.AuthConfig
			assert.NilError(t, json.Unmarshal(buf, &actualAuthConfig))
			assert.DeepEqual(t, actualAuthConfig, authConfig)
			fmt.Fprint(w, `{"token":"hub-content"}`)

		case hub.ScanTokenURL:
			assert.Equal(t, r.Method, http.MethodGet)
			assert.Equal(t, r.Header.Get("Authorization"), "Bearer hub-content")
			fmt.Fprint(w, `XXXX.YYYY.ZZZZ`)

		default:
			t.FailNow()
		}
	}))
	defer ts.Close()

	authenticator := NewAuthenticator(jose.JSONWebKeySet{}, ts.URL)
	token, err := authenticator.negotiateScanIDToken(authConfig)
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

			authenticator := NewAuthenticator(jose.JSONWebKeySet{}, "")
			authenticator.tokensPath = dir.Join("tokens.json")

			authConfig := types.AuthConfig{Username: "hubUser2"}

			token := authenticator.getLocalToken(authConfig)
			assert.Equal(t, token, testCase.expected)
		})
	}
}

func TestUpdateLocalToken(t *testing.T) {
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

			authenticator := NewAuthenticator(jose.JSONWebKeySet{}, "")
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

func TestCheckTokenValidity(t *testing.T) {
	// Generate JWKS file containing the public key
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       privateKey.Public(),
				KeyID:     "key-id",
				Algorithm: string(jose.ES256),
				Use:       "sig",
			},
		},
	}

	// Generate JWT token
	sig := newSigner(t, privateKey, "key-id")
	now := time.Now()

	testCases := []struct {
		name          string
		expectedError string
		generateToken func() string
	}{
		{
			name:          "empty token",
			generateToken: func() string { return "" },
			expectedError: "empty token",
		},
		{
			name:          "malformed token",
			generateToken: func() string { return "malformed token" },
			expectedError: `invalid token`,
		},
		{
			name: "signature don't match",
			generateToken: func() string {
				otherPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				sig := newSigner(t, otherPrivateKey, "key-id")
				return generateToken(t, sig, now)
			},
			expectedError: "invalid token: signature does not match the content",
		},
		{
			name:          "unknown key",
			expectedError: "invalid token: key identifier does not match",
			generateToken: func() string {
				sig := newSigner(t, privateKey, "unknown-key-id")
				return generateToken(t, sig, now)
			},
		},
		{
			name:          "expired token",
			expectedError: "token has expired",
			generateToken: func() string {
				return generateToken(t, sig, time.Unix(0, 0))
			},
		},
		{
			name:          "expired token in the last minute window",
			expectedError: "token has expired",
			generateToken: func() string {
				return generateToken(t, sig, now.Add(-(59*time.Minute + 30*time.Second)))
			},
		},
		{
			name: "valid token",
			generateToken: func() string {
				return generateToken(t, sig, now)
			},
			expectedError: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			authenticator := NewAuthenticator(jwks, "")
			err := authenticator.checkTokenValidity(testCase.generateToken())
			if testCase.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, testCase.expectedError)
			}
		})
	}
}

func newSigner(t *testing.T, key crypto.PrivateKey, kid string) jose.Signer {
	t.Helper()
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").
		WithHeader("kid", kid))
	assert.NilError(t, err)
	return sig
}

func generateToken(t *testing.T, sig jose.Signer, issueDate time.Time) string {
	t.Helper()
	cl := jwt.Claims{
		IssuedAt: jwt.NewNumericDate(issueDate),
		Expiry:   jwt.NewNumericDate(issueDate.Add(1 * time.Hour)),
	}
	raw, err := jwt.Signed(sig).Claims(cl).CompactSerialize()
	assert.NilError(t, err)
	return raw
}
