package provider

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type snykProvider struct {
}

// NewSnykProvider returns a Snyk implementation of scan provider
func NewSnykProvider() Provider {
	return &snykProvider{}
}

func (s *snykProvider) Version() (string, error) {
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
