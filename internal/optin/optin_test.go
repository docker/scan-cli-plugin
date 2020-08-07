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

package optin

import (
	"bytes"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestAskForConsent(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		consent bool
	}{
		{
			name:    "empty input rejects consent",
			input:   "",
			consent: false,
		},
		{
			name:    "invalid input rejects consent",
			input:   "invalid",
			consent: false,
		},
		{
			name:    "Upper case YES accepts consent",
			input:   "YES",
			consent: true,
		},
		{
			name:    "Lower case yes accepts consent",
			input:   "yes",
			consent: true,
		},
		{
			name:    "just y accepts consent",
			input:   "y",
			consent: true,
		},
		{
			name:    "spaces are ignored",
			input:   "   yes   ",
			consent: true,
		},
		{
			name:    "n rejects consent",
			input:   "n",
			consent: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stdin, stdout := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
			fmt.Fprint(stdin, testCase.input)
			consent := AskForConsent(stdin, stdout)
			assert.Equal(t, consent, testCase.consent)
			assert.Equal(t, stdout.String(), `Docker Scan relies upon access to Snyk, a third party provider, do you consent to proceed using Snyk? (y/N)
`)
		})
	}
}
