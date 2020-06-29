include vars.mk

NULL := /dev/null

ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif

ifeq ($(TAG_NAME),)
  TAG_NAME := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif

GOOS ?= $(shell go env GOOS)

PKG_NAME=github.com/docker/scan-cli-plugin
STATIC_FLAGS= CGO_ENABLED=0
LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG_NAME)"
GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)
BINARY:=docker-scan
SNYK_DOWNLOAD_NAME:=snyk-linux
SNYK_BINARY:=snyk
PWD:=$(shell pwd)
ifeq ($(GOOS),windows)
	BINARY=docker-scan.exe
	SNYK_DOWNLOAD_NAME:=snyk-win.exe
	SNYK_BINARY=snyk.exe
	PWD=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
endif
ifeq ($(GOOS),darwin)
	SNYK_DOWNLOAD_NAME:=snyk-macos
endif

VARS:= SNYK_DESKTOP_VERSION=${SNYK_DESKTOP_VERSION}\
	SNYK_USER_VERSION=${SNYK_USER_VERSION}\
	DOCKER_CONFIG=$(PWD)/docker-config\
	SNYK_USER_PATH=$(PWD)/docker-config/snyk-user\
	SNYK_DESKTOP_PATH=$(PWD)/docker-config/snyk-desktop

.PHONY: lint
lint:
	golangci-lint run --timeout 10m0s ./...

.PHONY: e2e
e2e:
	mkdir -p docker-config/scan
	mkdir -p docker-config/cli-plugins
	cp ./bin/${BINARY} docker-config/cli-plugins/${BINARY}
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

# For multi-platform (windows,macos,linux) github actions
.PHONY: download
download:
	mkdir -p docker-config/snyk-user
	curl https://github.com/snyk/snyk/releases/download/v${SNYK_USER_VERSION}/${SNYK_DOWNLOAD_NAME} -L -s -S -o docker-config/snyk-user/${SNYK_BINARY}
	chmod +x docker-config/snyk-user/${SNYK_BINARY}

	mkdir -p docker-config/snyk-desktop
	curl https://github.com/snyk/snyk/releases/download/v${SNYK_DESKTOP_VERSION}/${SNYK_DOWNLOAD_NAME} -L -s -S -o docker-config/snyk-desktop/${SNYK_BINARY}
	chmod +x docker-config/snyk-desktop/${SNYK_BINARY}
	
	GO111MODULE=on go get gotest.tools/gotestsum@v0.4.2
