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
