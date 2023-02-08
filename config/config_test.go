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

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
)

func TestSaveConfigFile(t *testing.T) {
	configDir, err := os.MkdirTemp(os.TempDir(), "config")
	assert.NilError(t, err)
	defer os.RemoveAll(configDir) //nolint:errcheck

	configFilePath := filepath.Join(configDir, "config.json")
	dockerConfig := dockerConfigFile.ConfigFile{
		CLIPluginsExtraDirs: []string{
			"cli-plugins",
		},
		Filename: configFilePath,
	}
	configFile, err := os.Create(configFilePath)
	assert.NilError(t, err)
	//nolint:errcheck
	defer configFile.Close()
	err = json.NewEncoder(configFile).Encode(dockerConfig)
	if err != nil {
		panic(err)
	}

	defer env.Patch(t, "DOCKER_CONFIG", configDir)

	expected := Config{
		Path:  configDir,
		Optin: false,
	}
	assert.NilError(t, SaveConfigFile(expected))

	result, err := ReadConfigFile()
	assert.NilError(t, err)
	assert.Equal(t, result, expected)
}
