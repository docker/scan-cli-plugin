export DOCKER_BUILDKIT=1

NULL := /dev/null
SNYK_DESKTOP_VERSION := 1.332.0
SNYK_USER_VERSION := 1.334.0
GO_VERSION := 1.14.3
CLI_VERSION := 19.03.9
BUILD_ARGS := --build-arg SNYK_DESKTOP_VERSION=$(SNYK_DESKTOP_VERSION)\
	--build-arg SNYK_USER_VERSION=$(SNYK_USER_VERSION)\
	--build-arg GO_VERSION=$(GO_VERSION)\
	--build-arg CLI_VERSION=$(CLI_VERSION)

ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif

ifeq ($(TAG_NAME),)
  TAG_NAME := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif

PKG_NAME=github.com/docker/docker-scan
STATIC_FLAGS= CGO_ENABLED=0
LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG_NAME)"

GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)
BINARY=docker-scan

.PHONY: build
build: ## Build docker-scan
	mkdir -p bin
	$(GO_BUILD) -o bin/$(BINARY) .

.PHONY: dbuild
dbuild: ## Build docker-scan in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./bin \
	--target scan

.PHONY: cross
cross: ## Cross compile docker-scan binaries in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./dist \
	--target cross

dist: ## Build cross compiled docker-scan binaries
	GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_linux_amd64/docker-scan/$(BINARY) .
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_darwin_amd64/docker-scan/$(BINARY) .
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/docker-scan_windows_amd64/docker-scan/$(BINARY).exe .

.PHONY: install
install: build ## Install docker-scan to your local cli-plugins directory
	cp bin/docker-scan ~/.docker/cli-plugins

.PHONY: e2e-build
e2e-build: ## Build e2e docker image 
	docker build $(BUILD_ARGS) . --target e2e -t docker-scan:e2e

.PHONY: e2e
e2e: e2e-build
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docker-scan:e2e

.PHONY: e2e-tests
e2e-tests:
	go test -ldflags=$(LDFLAGS) ./e2e

help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
