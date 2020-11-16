# Pinned Versions
SNYK_DESKTOP_VERSION=1.421.1
SNYK_USER_VERSION=1.421.0
SNYK_OLD_VERSION=1.382.1
GO_VERSION=1.15.0
CLI_VERSION=19.03.9
ALPINE_VERSION=3.12.0
GOLANGCI_LINT_VERSION=v1.27.0-alpine
GOTESTSUM_VERSION=0.5.2

GOOS ?= $(shell go env GOOS)
BINARY_EXT=
ifeq ($(GOOS),windows)
	BINARY_EXT=.exe
endif
PLATFORM_BINARY:=docker-scan_$(GOOS)_amd64$(BINARY_EXT)
BINARY=docker-scan$(BINARY_EXT)
