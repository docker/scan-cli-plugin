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

package internal

import (
	"fmt"
	"strings"

	"github.com/docker/scan-cli-plugin/internal/provider"
)

var (
	// Version is the version tag of the docker scan binary, set at build time
	Version = "unknown"
	// GitCommit is the commit of the docker scan binary, set at build time
	GitCommit = "unknown"
)

// FullVersion return plugin version, git commit and the provider cli version
func FullVersion(scanProvider provider.Provider) (string, error) {
	providerVersion, err := scanProvider.Version()
	if err != nil {
		return "", err
	}
	res := []string{
		fmt.Sprintf("Version:    %s", Version),
		fmt.Sprintf("Git commit: %s", GitCommit),
		fmt.Sprintf("Provider:   %s", providerVersion),
	}

	return strings.Join(res, "\n"), nil
}
