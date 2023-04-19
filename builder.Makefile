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
  -X $(PKG_NAME)/internal.Version=$(TAG_NAME)"
GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)

.PHONY: lint
lint:
	golangci-lint run --timeout 10m0s ./...

cross:
	GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_linux_amd64 ./cmd/docker-scan
	GOOS=linux   GOARCH=arm64 $(GO_BUILD) -o dist/docker-scan_linux_arm64 ./cmd/docker-scan
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_darwin_amd64 ./cmd/docker-scan
	GOOS=darwin  GOARCH=arm64 $(GO_BUILD) -o dist/docker-scan_darwin_arm64 ./cmd/docker-scan
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_windows_amd64.exe ./cmd/docker-scan

build-mac-arm64:
	mkdir -p bin
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o bin/docker-scan_darwin_arm64 ./cmd/docker-scan

build-linux-arm64:
	mkdir -p bin
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o bin/docker-scan_linux_arm64 ./cmd/docker-scan

.PHONY: build
build:
	mkdir -p bin
	$(GO_BUILD) -o bin/$(PLATFORM_BINARY) ./cmd/docker-scan
