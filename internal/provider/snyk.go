package provider

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type snykProvider struct {
	path string
}

// NewSnykProvider returns a Snyk implementation of scan provider
func NewSnykProvider(path string) Provider {
	return &snykProvider{path}
}

func (s *snykProvider) Version() (string, error) {
	cmd := exec.Command(s.path, "--version")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	if err := cmd.Run(); err != nil {
		if err == exec.ErrNotFound {
			// Could not find Snyk in $PATH
			return "", fmt.Errorf("could not find Snyk binary")
		} else if _, ok := err.(*os.PathError); ok {
			// The specified path for Snyk binary does not exist
			return "", fmt.Errorf("could not find Snyk binary")
		}
		return "", fmt.Errorf("fail to call Snyk: %s %s", err, buffErr.String())
	}
	return fmt.Sprintf("Snyk (%s)", strings.TrimSpace(buff.String())), nil
}
