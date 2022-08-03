include vars.mk

NULL := /dev/null

ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif

ifeq ($(TAG_NAME),)
  TAG_NAME := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif

PKG_NAME=github.com/docker/scan-cli-plugin
STATIC_FLAGS= CGO_ENABLED=0
LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG_NAME) \
  -X $(PKG_NAME)/internal/provider.ImageDigest=$(SNYK_IMAGE_DIGEST) \
  -X $(PKG_NAME)/internal/provider.SnykDesktopVersion=$(SNYK_DESKTOP_VERSION)"
GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)

SNYK_DOWNLOAD_NAME:=snyk-linux
SNYK_BINARY:=snyk
PWD:=$(shell pwd)
ifeq ($(GOOS),windows)
	SNYK_DOWNLOAD_NAME:=snyk-win.exe
	SNYK_BINARY=snyk.exe
	PWD=$(subst \,/,$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST)))))
endif
ifeq ($(GOOS),darwin)
	SNYK_DOWNLOAD_NAME:=snyk-macos
endif

ifneq ($(strip $(E2E_TEST_NAME)),)
	RUN_TEST=-test.run $(E2E_TEST_NAME)
endif

VARS:= SNYK_DESKTOP_VERSION=${SNYK_DESKTOP_VERSION}\
	SNYK_USER_VERSION=${SNYK_USER_VERSION}\
	SNYK_OLD_VERSION=${SNYK_OLD_VERSION}\
	DOCKER_CONFIG=$(PWD)/docker-config\
	SNYK_OLD_PATH=$(PWD)/docker-config/snyk-old\
	SNYK_USER_PATH=$(PWD)/docker-config/snyk-user\
	SNYK_DESKTOP_PATH=$(PWD)/docker-config/snyk-desktop

.PHONY: lint
lint:
	golangci-lint run --timeout 10m0s ./...

.PHONY: e2e
e2e:
	mkdir -p docker-config/scan
	mkdir -p docker-config/cli-plugins
	cp ./bin/${PLATFORM_BINARY} docker-config/cli-plugins/${BINARY}
	# TODO: gotestsum doesn't forward ldflags to go test with golang 1.15.0, so moving back to go test temporarily
	$(VARS) go test ./e2e $(RUN_TEST) -ldflags=$(LDFLAGS)

.PHONY: test-unit
test-unit:
	gotestsum $(shell go list ./... | grep -vE '/e2e')

cross:
	GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_linux_amd64 ./cmd/docker-scan
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_darwin_amd64 ./cmd/docker-scan
	GOOS=darwin  GOARCH=arm64 $(GO_BUILD) -o dist/docker-scan_darwin_arm64 ./cmd/docker-scan
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_windows_amd64.exe ./cmd/docker-scan

build-mac-arm64:
	mkdir -p bin
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o bin/docker-scan_darwin_arm64 ./cmd/docker-scan

.PHONY: build
build:
	mkdir -p bin
	$(GO_BUILD) -o bin/$(PLATFORM_BINARY) ./cmd/docker-scan

# For multi-platform (windows,macos,linux) github actions
.PHONY: download
download:
	mkdir -p docker-config/snyk-user
	curl https://github.com/snyk/snyk/releases/download/v${SNYK_USER_VERSION}/${SNYK_DOWNLOAD_NAME} -L -s -S -o docker-config/snyk-user/${SNYK_BINARY}
	chmod +x docker-config/snyk-user/${SNYK_BINARY}

	mkdir -p docker-config/snyk-old
	curl https://github.com/snyk/snyk/releases/download/v${SNYK_OLD_VERSION}/${SNYK_DOWNLOAD_NAME} -L -s -S -o docker-config/snyk-old/${SNYK_BINARY}
	chmod +x docker-config/snyk-old/${SNYK_BINARY}

	mkdir -p docker-config/snyk-desktop
	curl https://github.com/snyk/snyk/releases/download/v${SNYK_DESKTOP_VERSION}/${SNYK_DOWNLOAD_NAME} -L -s -S -o docker-config/snyk-desktop/${SNYK_BINARY}
	chmod +x docker-config/snyk-desktop/${SNYK_BINARY}
	
	GO111MODULE=on go get gotest.tools/gotestsum@v${GOTESTSUM_VERSION}
