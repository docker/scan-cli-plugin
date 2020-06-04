NULL := /dev/null

ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif

ifeq ($(TAG_NAME),)
  TAG_NAME := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif

GOOS ?= $(shell go env GOOS)

PKG_NAME=github.com/docker/docker-scan
STATIC_FLAGS= CGO_ENABLED=0
LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG_NAME)"
VARS:= SNYK_DESKTOP_VERSION=1.332.0 SNYK_USER_VERSION=1.334.0
GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)
BINARY:=docker-scan
ifeq ($(GOOS),windows)
	BINARY=docker-scan.exe
endif

.PHONY: lint
lint:
	golangci-lint run --timeout 10m0s ./...

.PHONY: e2e
e2e:
	$(VARS) gotestsum ./e2e -- -ldflags=$(LDFLAGS)

.PHONY: test-unit
test-unit:
	gotestsum $(shell go list ./... | grep -vE '/e2e')

cross:
	GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_linux_amd64 ./cmd/docker-scan
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_darwin_amd64 ./cmd/docker-scan
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_windows_amd64.exe ./cmd/docker-scan

.PHONY: build
build:
	mkdir -p bin
	$(GO_BUILD) -o bin/$(BINARY) ./cmd/docker-scan
