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
	"bufio"
	"fmt"
	"io"
	"strings"
)

// AskForConsent prompts a consent question to inform about Snyk usage on behalf
func AskForConsent(stdin io.Reader, stdout io.Writer) bool {
	fmt.Fprintln(stdout, "Docker Scan relies upon access to Snyk, a third party provider, do you consent to proceed using Snyk? (y/N)")
	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	switch input {
	case "", "n", "no":
		return false
	case "y", "yes":
		return true
	default: // anything else reject the consent
		return false
	}
}
