include vars.mk
export DOCKER_BUILDKIT=1

BUILD_ARGS := --build-arg SNYK_DESKTOP_VERSION=$(SNYK_DESKTOP_VERSION)\
	--build-arg SNYK_USER_VERSION=$(SNYK_USER_VERSION)\
	--build-arg GO_VERSION=$(GO_VERSION)\
	--build-arg CLI_VERSION=$(CLI_VERSION)\
	--build-arg ALPINE_VERSION=$(ALPINE_VERSION)\
	--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
	--build-arg TAG_NAME=$(GIT_TAG_NAME) \
	--build-arg GOTESTSUM_VERSION=$(GOTESTSUM_VERSION)

E2E_ENV := --env E2E_TEST_AUTH_TOKEN \
           --env E2E_HUB_URL \
           --env E2E_HUB_USERNAME \
           --env E2E_HUB_TOKEN \
           --env E2E_TEST_NAME

.PHONY: all
all: lint build test

.PHONY: build
build: ## Build docker-scan in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./bin \
	--platform local \
	--target scan

.PHONY: cross
cross: ## Cross compile docker-scan binaries in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./dist \
	--target cross

.PHONY: install
install: build ## Install docker-scan to your local cli-plugins directory
	cp bin/docker-scan ~/.docker/cli-plugins

.PHONY: test ## Run unit tests then end-to-end tests
test: test-unit e2e

.PHONY: e2e-build
e2e-build:
	docker build $(BUILD_ARGS) . --target e2e -t docker-scan:e2e

.PHONY: e2e
e2e: e2e-build ## Run the end-to-end tests
	@docker run $(E2E_ENV) --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(shell go env GOCACHE):/root/.cache/go-build docker-scan:e2e

test-unit-build:
	docker build $(BUILD_ARGS) . --target test-unit -t docker-scan:test-unit

.PHONY: test-unit
test-unit: test-unit-build ## Run unit tests
	docker run --rm -v $(shell go env GOCACHE):/root/.cache/go-build docker-scan:test-unit

.PHONY: lint
lint: ## Run the go linter
	@docker build . --target lint

.PHONY: validate
validate: ## Validate files license header
	docker run --rm -v $(CURDIR):/work -w /work \
	 golang:${GO_VERSION} \
	 bash -c 'go get -u github.com/kunalkushwaha/ltag && ./scripts/validate/fileheader'

.PHONY: help
help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
