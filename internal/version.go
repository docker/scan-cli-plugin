package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

var (
	// Version is the version tag of the docker scan binary, set at build time
	Version = "unknown"
	// GitCommit is the commit of the docker scan binary, set at build time
	GitCommit = "unknown"
)

// FullVersion return plugin version, git commit and the provider cli version
func FullVersion() (string, error) {
	provider, err := providerVersion()
	if err != nil {
		return "", err
	}
	res := []string{
		fmt.Sprintf("Version:    %s", Version),
		fmt.Sprintf("Git commit: %s", GitCommit),
		fmt.Sprintf("Provider:   %s", provider),
	}

	return strings.Join(res, "\n"), nil
}

func providerVersion() (string, error) {
	cmd := exec.Command("snyk", "--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fail to call Snyk: %s", buffErr.String())
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}
