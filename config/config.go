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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	cliConfig "github.com/docker/cli/cli/config"
)

// Config points to scan provider's binary
type Config struct {
	Path  string `json:"path"`
	Optin bool   `json:"optin"`
}

// ReadConfigFile tries to read docker-scan configuration file that
// should be at ${DOCKER_CONFIG}/scan/config.json
func ReadConfigFile() (Config, error) {
	var conf Config
	path := filepath.Join(cliConfig.Dir(), "scan", "config.json")
	buf, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return conf, nil
	}
	if err != nil {
		return conf, errors.Wrap(err, "failed to read docker scan configuration file")
	}
	if err := json.Unmarshal(buf, &conf); err != nil {
		return conf, errors.Wrapf(err, "invalid docker scan configuration file %s", path)
	}
	return conf, nil
}

// SaveConfigFile tries to save docker-scan configuration file that
// should be at ${DOCKER_CONFIG}/scan/config.json
func SaveConfigFile(conf Config) error {
	out, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(cliConfig.Dir(), "scan"), 0744); err != nil {
		return errors.Wrap(err, "failed to create docker scan configuration directory")
	}

	path := filepath.Join(cliConfig.Dir(), "scan", "config.json")
	return errors.Wrap(ioutil.WriteFile(path, out, os.FileMode(644)), "failed to write docker scan configuration file")
}
