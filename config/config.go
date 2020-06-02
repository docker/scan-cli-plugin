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
	Path string `json:"path"`
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
